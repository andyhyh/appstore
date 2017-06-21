package debug

import (
	"github.com/Sirupsen/logrus"
	"time"
)

func GetFunctionTiming(t1 time.Time, msg string, fields logrus.Fields, logger *logrus.Entry) {
	t2 := time.Now()
	fields["took_ns"] = t2.Sub(t1)
	logger.WithFields(fields).Debug(msg)
}
