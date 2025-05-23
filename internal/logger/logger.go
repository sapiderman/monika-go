// Package logger initializes logger feature
package logger

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

func InitLogger() {
	// Set the log level to info
	logrus.SetLevel(logrus.DebugLevel)

	// Set the log format to JSON
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		PrettyPrint:     false,
	})

	// Set the output to stdout
	logrus.SetOutput(os.Stdout)
}
