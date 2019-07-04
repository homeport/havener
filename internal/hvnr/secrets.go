// Copyright © 2018 The Havener
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package hvnr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/neat"
	"github.com/gonvenience/wait"
	"github.com/gonvenience/wrap"
	"github.com/homeport/havener/pkg/havener"
	colorful "github.com/lucasb-eyer/go-colorful"
	zxcvbn "github.com/nbutton23/zxcvbn-go"
)

type pwd struct {
	namespace string
	secret    string
	name      string
	cracktime float64
	score     int
}

// PrintSecretsAnalysis prints an analysis of the secrets in the cluster
func PrintSecretsAnalysis() error {
	client, _, err := havener.OutOfClusterAuthentication("")
	if err != nil {
		return wrap.Errorf(err, "failed to access cluster")
	}

	pi := wait.NewProgressIndicator("Scanning all secrets in cluster ...")
	pi.Start()

	namespaces, err := havener.ListNamespaces(client)
	if err != nil {
		return err
	}

	list := []pwd{}

	for _, namespace := range namespaces {
		if namespace == "kube-system" {
			continue
		}

		secrets, err := havener.SecretsInNamespace(client, namespace)
		if err != nil {
			return err
		}

		for _, secret := range secrets {
			switch secret.Type {
			case "Opaque":
				for name, data := range secret.Data {
					value := string(data)

					if len(data) == 0 ||
						strings.HasSuffix(name, ".generator") ||
						strings.HasPrefix(value, "-----BEGIN") ||
						strings.Contains(name, "fingerprint") {
						continue
					}

					score := zxcvbn.PasswordStrength(value, []string{})
					list = append(list, pwd{
						namespace: namespace,
						secret:    secret.Name,
						name:      name,
						cracktime: score.CrackTime,
						score:     score.Score,
					})
				}

			case "kubernetes.io/dockerconfigjson":
				if dockerconfigjson, ok := secret.Data[".dockerconfigjson"]; ok {
					endpoint, password, err := extractPasswordFromDockerConfigJSON(dockerconfigjson)
					if err != nil {
						return err
					}

					score := zxcvbn.PasswordStrength(password, []string{})
					list = append(list, pwd{
						namespace: namespace,
						secret:    secret.Name,
						name:      endpoint,
						cracktime: score.CrackTime,
						score:     score.Score,
					})
				}

			case "kubernetes.io/dockercfg":
				if dockercfg, ok := secret.Data[".dockercfg"]; ok {
					var authsMap map[string]interface{}
					if err := json.Unmarshal(dockercfg, &authsMap); err != nil {
						return err
					}

					endpoint, password, err := extractPasswordFromDockerCfg(authsMap)
					if err != nil {
						return err
					}

					score := zxcvbn.PasswordStrength(password, []string{})
					list = append(list, pwd{
						namespace: namespace,
						secret:    secret.Name,
						name:      endpoint,
						cracktime: score.CrackTime,
						score:     score.Score,
					})
				}
			}
		}
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].score != list[j].score {
			return list[i].score < list[j].score
		}

		if list[i].namespace != list[j].namespace {
			return list[i].namespace < list[j].namespace
		}

		if list[i].secret != list[j].secret {
			return list[i].secret < list[j].secret
		}

		if list[i].name != list[j].name {
			return list[i].name < list[j].name
		}

		return list[i].cracktime < list[j].cracktime
	})

	result := [][]string{
		[]string{
			bunt.Sprintf("*namespace*"),
			bunt.Sprintf("*secret*"),
			bunt.Sprintf("*name*"),
			bunt.Sprintf("*score*"),
		},
	}

	for _, pwd := range list {
		result = append(result, []string{
			pwd.namespace,
			pwd.secret,
			pwd.name,
			pwd.Strength(),
		})
	}

	pi.Stop()

	output, err := neat.Table(result)
	if err != nil {
		return err
	}

	neat.Box(os.Stdout,
		"Scores of passwords found in Cluster secrets",
		strings.NewReader(output),
		neat.HeadlineColor(bunt.SteelBlue),
	)

	return nil
}

func (p *pwd) Strength() string {
	var (
		buf   bytes.Buffer
		text  string
		color colorful.Color
	)

	switch p.score {
	case 0:
		color = bunt.Red
		text = "inacceptable"

	case 1:
		color = bunt.OrangeRed
		text = "weak"

	case 2:
		color = bunt.Goldenrod
		text = "moderate"

	case 3:
		color = bunt.DarkGreen
		text = "strong"

	case 4:
		color = bunt.Green
		text = "very strong"
	}

	coloredBoxes := p.score
	grayBoxes := 4 - p.score

	for i := 0; i < coloredBoxes; i++ {
		fmt.Fprint(&buf, bunt.Style("◼◼ ", bunt.Foreground(color)))
	}

	for i := 0; i < grayBoxes; i++ {
		fmt.Fprint(&buf, bunt.Style("◼◼ ", bunt.Foreground(bunt.DimGray)))
	}

	fmt.Fprint(&buf, bunt.Style(text, bunt.Foreground(color)))

	return buf.String()
}

func extractPasswordFromDockerConfigJSON(data []byte) (string, string, error) {
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", "", err
	}

	auths, ok := cfg["auths"]
	if !ok {
		return "", "", fmt.Errorf("failed to find 'auths' in Docker configuration JSON")
	}

	authsMap, ok := auths.(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("failed to match 'auths' value to type map")
	}

	return extractPasswordFromDockerCfg(authsMap)
}

func extractPasswordFromDockerCfg(authsMap map[string]interface{}) (string, string, error) {
	for endpoint, entry := range authsMap {
		switch settings := entry.(type) {
		case map[string]interface{}:
			if password, ok := settings["password"]; ok {
				result, ok := password.(string)
				if !ok {
					return "", "", fmt.Errorf("failed to match password to type string")
				}

				return endpoint, result, nil
			}

		default:
			return "", "", fmt.Errorf("failed to match settings to type map, type is %T", settings)
		}
	}

	return "", "", fmt.Errorf("failed to find password in given config")
}
