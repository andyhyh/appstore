package debug

import (
	log "github.com/Sirupsen/logrus"
	"time"
)

func GetFunctionTiming(t1 time.Time, msg string, fields log.Fields) {
	t2 := time.Now()
	fields["took_ns"] = t2.Sub(t1)
	log.WithFields(fields).Debug(msg)
}
