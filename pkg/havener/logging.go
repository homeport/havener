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
