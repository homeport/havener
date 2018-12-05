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
	"net/url"
	"os"
	"runtime/debug"
	"strings"

	"github.com/homeport/gonvenience/pkg/v1/bunt"
)

// NoUserPrompt defines whether a user confirmation is required or should be omitted
var NoUserPrompt = false

// PromptUser prompts the user via STDIN to confirm the message with either 'yes', or 'no' -- yes being translated into true, everything else is false.
func PromptUser(message string) bool {
	// Assume yes if the NoUserPrompt is set
	if NoUserPrompt {
		return true
	}

	fmt.Printf(message)

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
