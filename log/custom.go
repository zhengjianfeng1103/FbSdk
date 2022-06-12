package log

import (
	"github.com/sirupsen/logrus"
	"os"
)

var Log *logrus.Logger

func Init(level logrus.Level) {
	logger := logrus.New()
	logger.SetLevel(level)
	logger.SetOutput(os.Stdout)
	Log = logger
}
