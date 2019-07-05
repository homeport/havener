package cmd

import (
	"os"

	"github.com/gonvenience/wait"

	"github.com/gonvenience/bunt"

	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/viper"
)

var progressIndicator *wait.ProgressIndicator

// LogTask processes all log messages coming
// from havener package.
func LogTask(signals chan os.Signal) {
	channel := havener.GetLogChannel()

	for {
		select {
		case message := <-channel:
			targetLevel := translateLogLevel()
			log(message, targetLevel)
		case _ = <-signals:
			close(channel)
			return
		}
	}
}

// setCurrentProgressIndicator updates the gobal variable
// which is used for text updates of the current indicator.
// For resetting use nil value.
func setCurrentProgressIndicator(pi *wait.ProgressIndicator) {
	progressIndicator = pi
}

// translateLogLevel transates the flag boolean to
// the associated log level.
// Levels: Fatal < Error < Warn < Verbose < Debug < Trace
func translateLogLevel() havener.LogLevel {
	logLevel := havener.Off

	fatalLevel := viper.GetBool("fatal")
	errorLevel := viper.GetBool("error")
	warnLevel := viper.GetBool("warn")
	verboseLevel := viper.GetBool("verbose")
	debugLevel := viper.GetBool("debug")
	traceLevel := viper.GetBool("trace")

	if fatalLevel && havener.Fatal > logLevel {
		logLevel = havener.Fatal
	}
	if errorLevel && havener.Error > logLevel {
		logLevel = havener.Error
	}
	if warnLevel && havener.Warn > logLevel {
		logLevel = havener.Warn
	}
	if verboseLevel && havener.Verbose > logLevel {
		logLevel = havener.Verbose
	}
	if debugLevel && havener.Debug > logLevel {
		logLevel = havener.Debug
	}
	if traceLevel && havener.Trace > logLevel {
		logLevel = havener.Trace
	}

	return logLevel
}

// log processes all log messages and logs them differently
// according to their level
func log(message havener.LogMessage, targetLevel havener.LogLevel) {
	if targetLevel >= message.Level {
		switch message.Level {
		case havener.Fatal:
			printLogf("*[FATAL]* %s\n", message.Message)
		case havener.Error:
			printLogf("*[ERROR]* %s\n", message.Message)
		case havener.Warn:
			printLogf("*[WARN]* %s\n", message.Message)
		case havener.Verbose:
			printLogf("*[INFO]* %s\n", message.Message)
		case havener.Debug:
			printLogf("*[DEBUG]* %s\n", message.Message)
		case havener.Trace:
			printLogf("*[TRACE]* %s\n", message.Message)
		default:
			printLogf("*[INFO]* %s\n", message.Message)
		}
	}
}

// printLogf formats and prints a log message
func printLogf(format string, args ...interface{}) {
	if progressIndicator != nil {
		progressIndicator.SetText(format, args...)
	} else {
		bunt.Printf(format, args...)
	}
}
