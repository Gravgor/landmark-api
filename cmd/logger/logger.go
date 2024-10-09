package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

func init() {
	Logger = logrus.New()

	// Set log output to a file
	logFile, err := os.OpenFile("api.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		Logger.Fatal(err)
	}

	Logger.SetOutput(logFile)
	Logger.SetFormatter(&logrus.JSONFormatter{}) // Use JSON format for structured logs
	Logger.SetLevel(logrus.InfoLevel)            // Set the default log level
}

// LogEvent logs structured events
func LogEvent(level logrus.Level, message string, fields logrus.Fields) {
	Logger.WithFields(fields).Log(level, message)
}
