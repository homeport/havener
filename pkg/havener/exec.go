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

package havener

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/gonvenience/wrap"
)

// ProcessConfigFile reads the havener config file from the provided path and
// returns the processed input file. Any shell operator will be evaluated.
func ProcessConfigFile(path string) (*Config, error) {

	config, err := ParseConfigFile(path)
	if err != nil {
		return nil, err
	}

	err = SetConfigEnv(config)
	if err != nil {
		return nil, err
	}

	for idx, release := range config.Releases {
		config.Releases[idx].ChartName, err = ProcessOperators(release.ChartName)
		if err != nil {
			return nil, err
		}
		config.Releases[idx].ChartVersion, err = ProcessOperators(release.ChartVersion)
		if err != nil {
			return nil, err
		}
		config.Releases[idx].ChartNamespace, err = ProcessOperators(release.ChartNamespace)
		if err != nil {
			return nil, err
		}
		config.Releases[idx].ChartLocation, err = ProcessOperators(release.ChartLocation)
		if err != nil {
			return nil, err
		}

		if release.Overrides == nil {
			continue
		}
		config.Releases[idx].Overrides, err = TraverseStructureAndProcessOperators(release.Overrides)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

// TraverseStructureAndProcessOperators traverses the provided generic structure
// and processes all string leafs.
func TraverseStructureAndProcessOperators(input interface{}) (interface{}, error) {
	var err error

	switch obj := input.(type) {
	case map[interface{}]interface{}:
		for key, value := range obj {
			obj[key], err = TraverseStructureAndProcessOperators(value)
			if err != nil {
				return nil, err
			}
		}

	case []interface{}:
		for idx, value := range obj {
			obj[idx], err = TraverseStructureAndProcessOperators(value)
			if err != nil {
				return nil, err
			}
		}

	case string:
		input, err = ProcessOperators(obj)
		if err != nil {
			return nil, err
		}

		if isPotentialBoolean(input) {
			input, _ = strconv.ParseBool(input.(string))
		}

	case nil:
		input, err = map[interface{}]interface{}{}, nil
	}

	return input, err
}

// isPotentialBoolean checks if the argument looks like
// a false/true bool. This is not supported by default
// in shell env variables. But, is a way to allow users
// to define booleans in the values.yml, where booleans
// are supported by helm
func isPotentialBoolean(input interface{}) bool {
	shellRegexp := regexp.MustCompile(`^(false|true)$`)
	switch obj := input.(type) {
	case string:
		if matches := shellRegexp.FindAllStringSubmatch(obj, -1); len(matches) > 0 {
			return true
		}
	}
	return false
}

// ProcessOperators processes the input string and calls
// all operator checks.
func ProcessOperators(s string) (string, error) {
	var err error

	inputString := s

	inputString, err = processSecretOperator(inputString)
	if err != nil {
		return "", err
	}

	inputString, err = processShellOperator(inputString)
	if err != nil {
		return "", err
	}

	inputString = processEnvOperator(inputString)

	return inputString, nil
}

// processShellOperator processes the input string and evaluates any shell
// operator in it.
func processShellOperator(s string) (string, error) {
	// https://regex101.com/r/dvdiTp/2
	shellRegexp := regexp.MustCompile(`\({2}\s*shell\s*(.+?)\s*\B\){2}`)
	if matches := shellRegexp.FindAllStringSubmatch(s, -1); len(matches) > 0 {
		for _, match := range matches {
			/* #0 is the whole string,
			 * #1 is the command
			 */
			cmd := exec.Command("sh", "-c", match[1])

			var out bytes.Buffer
			cmd.Stdout = &out

			if err := cmd.Run(); err != nil {
				return "", wrap.Errorf(err, "failed to run command: %s", match[1])
			}

			result := strings.TrimSpace(out.String())
			s = strings.Replace(s, match[0], result, -1)
		}
	}

	return s, nil
}

// processEnvOperator processes the input string and resolves any
// environment variable in it.
func processEnvOperator(s string) string {
	// https://regex101.com/r/SZ5CDH/2
	envRegexp := regexp.MustCompile(`\({2}\s*env\s*(([\w]+).+?)\s*\B\){2}`)
	if matches := envRegexp.FindAllStringSubmatch(s, -1); len(matches) > 0 {
		for _, match := range matches {
			/* #0 is the whole string,
			 * #1 env name
			 * #2 rest
			 */

			variableName := strings.TrimSpace(match[1])
			variable := os.Getenv(variableName)
			s = strings.Replace(s, match[0], variable, -1)
		}
	}

	return s
}

// processSecretOperator processes the input string and resolves any
// secret from the provided namespace, name and key in it.
func processSecretOperator(s string) (string, error) {
	// https://regex101.com/r/GnyRAa/2/
	secretRegexp := regexp.MustCompile(`\({2}\s*secret\s*(\S*)\s*(\S*)\s*(\S*)\s*\B\){2}`)
	if matches := secretRegexp.FindAllStringSubmatch(s, -1); len(matches) > 0 {
		for _, match := range matches {
			/* #0 is the whole string,
			 * #1 namespace
			 * #2 secret name
			 * #3 secret key
			 */

			namespace := fmt.Sprintf("%v", match[1])
			secretName := fmt.Sprintf("%v", match[2])
			secretKey := fmt.Sprintf("%v", match[3])

			if namespace == "" || secretName == "" || secretKey == "" {
				return "", fmt.Errorf("invalid arguments for secret operator")
			}

			secretValue, err := getSecretValue(namespace, secretName, secretKey)
			if err != nil {
				return "", err
			}

			s = strings.Replace(s, match[0], string(secretValue), -1)
		}
	}

	return s, nil
}
