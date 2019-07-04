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
// - Off:
// - Fatal:
// - Error:
// - Warn:
// - Info:
// - Debug:
// - Trace:
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

func logf(level LogLevel, message string, fArgs ...interface{}) {
	logChannel <- LogMessage{
		Message: fmt.Sprintf(message, fArgs...),
		Level:   level,
		Date:    time.Now(),
	}
}
