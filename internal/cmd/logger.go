package cmd

import (
	"os"

	"github.com/gonvenience/bunt"

	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/viper"
)

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

func translateLogLevel() havener.LogLevel {
	logLevel := havener.Off

	verbose := viper.GetBool("verbose")
	debug := viper.GetBool("debug")
	trace := viper.GetBool("trace")

	if verbose && havener.Verbose > logLevel {
		logLevel = havener.Verbose
	}
	if debug && havener.Debug > logLevel {
		logLevel = havener.Debug
	}
	if trace && havener.Trace > logLevel {
		logLevel = havener.Trace
	}

	return logLevel
}

func log(message havener.LogMessage, targetLevel havener.LogLevel) {
	if targetLevel >= message.Level {
		switch message.Level {
		case havener.Verbose:
			bunt.Printf("*[INFO]* %s\n", message.Message)
		case havener.Debug:
			bunt.Printf("*[DEBUG]* %s\n", message.Message)
		case havener.Trace:
			bunt.Printf("*[TRACE]* %s\n", message.Message)
		default:
			bunt.Printf("*[INFO]* %s\n", message.Message)
		}
	}
}
