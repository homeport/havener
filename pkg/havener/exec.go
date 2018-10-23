package havener

import (
	"regexp"
	"bytes"
	"io/ioutil"
	"os/exec"
	"strings"
	"fmt"
	yaml "gopkg.in/yaml.v2"
)

// ProcessConfigFile reads the havener config file from the provided path and
// returns the processed input file. Any shell operator will be evaluated.
func ProcessConfigFile(path string) (*Config, error) {
	source, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err = yaml.Unmarshal(source, &config); err != nil {
		return nil, err
	}

	for idx, release := range config.Releases {
		if release.Overrides == nil {
			continue
		}

		config.Releases[idx].Overrides, err = traverseStructureAndProcessShellOperators(release.Overrides)
		if err != nil {
			return nil, err
		}
	}

	return &config, nil
}

// traverseStructureAndProcessShellOperators traverses the provided generic structure
// and processes all string leafs.
func traverseStructureAndProcessShellOperators(input interface {}) (interface {}, error) {
	var err error

	switch input.(type) {
	case map[interface {}] interface {}:
		newmap := input.(map[interface {}] interface {})
		for key, value := range newmap {
			newmap[key], err = traverseStructureAndProcessShellOperators(value)
			if err != nil {
				return nil, err
			}
		}

	case []interface{}:
		newarr := input.([]interface{})
		for idx, value := range newarr {
			newarr[idx], err = traverseStructureAndProcessShellOperators(value)
			if err != nil {
				return nil, err
			}
		}

	case string:
		input, err = processShellOperator(input.(string))
		if err != nil {
			return nil, err
		}
	}

	return input, err
}

// processShellOperator processes the input string and evaluates any shell
// operator in it.
func processShellOperator(s string) (string, error) {
	// https://regex101.com/r/dvdiTp/1
	shellRegexp := regexp.MustCompile(`\(\(\s*shell\s+(.+)\s*\)\)`)
	if matches := shellRegexp.FindAllStringSubmatch(s, -1); len(matches) > 0 {
		for _, match := range matches {
			/* #0 is the whole string,
			 * #1 is the command
			 */
			cmd := exec.Command("sh", "-c", match[1])

			var out bytes.Buffer
			cmd.Stdout = &out

			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("failed to run command: %s\nerror message: %s", match[1], err.Error())
			}

			result := strings.TrimSpace(out.String())
			s = strings.Replace(s, match[0], result, -1)
		}
	}

	return s, nil
}
