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

package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"

	"github.com/gonvenience/bunt"
	"github.com/homeport/havener/pkg/havener"
	colorful "github.com/lucasb-eyer/go-colorful"
	yaml "gopkg.in/yaml.v2"
)

// ErrorWithMsg defines a custom msg together
// with an error
type ErrorWithMsg struct {
	Msg string
	Err error
}

func (e *ErrorWithMsg) Error() string {
	return fmt.Sprintf("%v\n", e.Err)
}

// NoUserPrompt defines whether a user confirmation is required or should be omitted
var NoUserPrompt = false

// PromptUser prompts the user via STDIN to confirm the message with either 'yes', or 'no' -- yes being translated into true, everything else is false.
func PromptUser(message string) bool {
	// Assume yes if the NoUserPrompt is set
	if NoUserPrompt {
		return true
	}

	fmt.Println(message)

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		switch strings.ToLower(scanner.Text()) {
		case "yes", "y":
			return true

		default:
			return false
		}
	}

	return false
}

// exitWithError leaves the tool with the provided error message
func exitWithError(msg string, err error) {
	bunt.Printf("Coral{*%s*}\n", msg)

	if err != nil {
		for _, line := range strings.Split(err.Error(), "\n") {
			bunt.Printf("Coral{│} DimGray{%s}\n", line)
		}
	}

	os.Exit(1)
}

// exitWithErrorAndIssue leaves the tool with the provided error message and a
// link that can be used to open a GitHub issue
func exitWithErrorAndIssue(msg string, err error) {
	bunt.Printf("Coral{*%s*}\n", msg)

	var errMsg = msg
	if err != nil {
		errMsg = err.Error()

		for _, line := range strings.Split(err.Error(), "\n") {
			bunt.Printf("Coral{│} DimGray{%s}\n", line)
		}
	}

	var buf bytes.Buffer
	buf.WriteString(errMsg)
	buf.WriteString("\n\nStacktrace:\n```")
	buf.WriteString(string(debug.Stack()))
	buf.WriteString("```")

	bunt.Printf("\nIf you like to open an issue in GitHub:\nCornflowerBlue{~https://github.com/homeport/havener/issues/new?title=%s&body=%s~}\n\n",
		url.PathEscape("Report panic: "+errMsg),
		url.PathEscape(buf.String()),
	)

	os.Exit(1)
}

func processTask(title string, task *havener.Task) error {
	if task == nil {
		return nil
	}

	data := make(chan string)
	defer close(data)
	go func() {
		streamStyledMessageInGray(title, data)
	}()

	for _, taskEntry := range *task {
		var cmd string
		var args []string
		var err error

		switch taskEntry := taskEntry.(type) {
		case string:
			cmd, args = "/bin/sh", append(args, "-c", taskEntry)

		case map[interface{}]interface{}:
			cmd, args, err = parseCommandFromMap(taskEntry)

		default:
			return fmt.Errorf("unsupported command specification (type %T):\n%v", taskEntry, taskEntry)
		}

		if err != nil {
			return err
		}

		read, write := io.Pipe()
		go func() {
			command := exec.Command(cmd, args...)
			command.Stdout = write
			command.Stderr = write
			err = command.Run()
			write.Close()
		}()

		scanner := bufio.NewScanner(read)
		for scanner.Scan() {
			data <- bunt.RemoveAllEscapeSequences(scanner.Text())
		}

		if err != nil {
			return fmt.Errorf("failed to run command: %s %s\n%s",
				cmd,
				strings.Join(args, " "),
				err.Error())
		}
	}

	return nil
}

func parseCommandFromMap(data map[interface{}]interface{}) (string, []string, error) {
	var command string
	var arguments []string

	cmd, ok := data["cmd"]
	if !ok {
		return "", nil, fmt.Errorf("failed to find mandatory entry 'cmd'")
	}

	switch cmd := cmd.(type) {
	case string:
		command = cmd

	default:
		return "", nil, fmt.Errorf("incompatible types, mandatory entry 'cmd' must be of type string")
	}

	if args, ok := data["args"]; ok {
		switch args.(type) {
		case []interface{}:
			for _, entry := range args.([]interface{}) {
				switch entry := entry.(type) {
				case string:
					arguments = append(arguments, entry)

				default:
					return "", nil, fmt.Errorf("incompatible types, the 'args' entries must be of type string")
				}
			}

		default:
			return "", nil, fmt.Errorf("incompatible types, optional entry 'args' must be of type list")
		}
	}

	return command, arguments, nil
}

func streamStyledMessageInGray(caption string, data chan string) {
	streamStyledMessage(caption, data, bunt.Gray, bunt.DimGray)
}

func streamStyledMessage(caption string, data chan string, captionColor colorful.Color, dataColor colorful.Color) {
	captionPrinted := false

	for line := range data {
		if !captionPrinted {
			bunt.Printf("*%s*\n", bunt.Style(caption, bunt.Foreground(captionColor)))
			captionPrinted = true
		}

		bunt.Printf("%s %s\n",
			bunt.Style("│", bunt.Foreground(captionColor)),
			bunt.Style(line, bunt.Foreground(dataColor)),
		)
	}

	if captionPrinted {
		bunt.Println()
	}
}

func printStatusMessage(head string, body string, headColor colorful.Color) {
	bunt.Printf("*%s*\n", bunt.Style(head, bunt.Foreground(headColor)))
	for _, line := range strings.Split(body, "\n") {
		fmt.Printf("%s %s\n",
			bunt.Style("│", bunt.Foreground(headColor)),
			line,
		)
	}

	bunt.Println()
}

// processOverrideSection rewrites the override section of the havener config
// by resolving its operators.
func processOverrideSection(release havener.Release) ([]byte, error) {
	overrides, err := havener.TraverseStructureAndProcessOperators(release.Overrides)
	if err != nil {
		return nil, &ErrorWithMsg{"failed to process overrides section", err}
	}

	overridesData, err := yaml.Marshal(overrides)
	if err != nil {
		return nil, &ErrorWithMsg{"failed to marshal overrides structure into bytes", err}
	}

	return overridesData, nil
}

// getReleaseMessage combines a custom message with the release notes
// from the helm binary.
func getReleaseMessage(release havener.Release, message string) (string, error) {
	releaseNotes, err := havener.RunHelmBinary("get", "notes", release.ChartName)
	if err != nil {
		return "", &ErrorWithMsg{"failed to get notes of release", err}
	}

	if releaseNotes != nil {
		message = message + "\n\n" + string(releaseNotes)
	}

	return message, nil
}
