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
	"bufio"
	"fmt"
	"os"
	"strings"
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
