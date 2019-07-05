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
	"fmt"
	"time"
)

// LogLevel indicates the priority level of the log message
// which can be used for filtering
type LogLevel int

// LogMessage is a helper structure for transmitting log messages
// to the logger
type LogMessage struct {
	Message string
	Level   LogLevel
	Date    time.Time
}

// Supported log levels include:
// - Off: don't print any logs
// - Fatal: only print fatal logs
// - Error: print error logs and previous levels
// - Warn: print warn logs and previous levels
// - Verbose: print verbose/info logs and previous levels
// - Debug: print debug logs and previous levels
// - Trace: print all logs
const (
	Off = LogLevel(iota)
	Fatal
	Error
	Warn
	Verbose
	Debug
	Trace
)

var logChannel = make(chan LogMessage, 100)

// GetLogChannel return the log channel to be used
// within the internal package.
func GetLogChannel() chan LogMessage {
	return logChannel
}

// logf formats the message and sends it to the logger
func logf(level LogLevel, message string, fArgs ...interface{}) {
	logChannel <- LogMessage{
		Message: fmt.Sprintf(message, fArgs...),
		Level:   level,
		Date:    time.Now(),
	}
}
