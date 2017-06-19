package logger

import (
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pressly/chi/middleware"
)

var debug = false

const httpProtoMajor = 1

func SetDebug(enableDebug bool) {
	debug = enableDebug
}

func Logger(next http.Handler) http.Handler {
	log := logrus.New()
	log.Level = logrus.InfoLevel
	if !debug {
		log.Formatter = &logrus.JSONFormatter{}
	}

	fn := func(w http.ResponseWriter, r *http.Request) {
		user := "unknown"
		reqID := middleware.GetReqID(r.Context())
		entry := extractFromReq(log, reqID, user, r)
		lw := middleware.NewWrapResponseWriter(w, httpProtoMajor)

		t1 := time.Now()
		defer func() {
			t2 := time.Now()
			logRequest(log, entry, lw, t2.Sub(t1))
		}()

		next.ServeHTTP(lw, r)
	}

	return http.HandlerFunc(fn)
}

func extractFromReq(log *logrus.Logger, reqID string, user string, r *http.Request) *logrus.Entry {
	entry := logrus.NewEntry(log).WithFields(logrus.Fields{
		"reqID":     reqID,
		"method":    r.Method,
		"host":      r.Host,
		"uri":       r.RequestURI,
		"proto":     r.Proto,
		"remote_ip": r.RemoteAddr,
		"user":      user,
		"ssl":       r.TLS != nil,
	})

	if r.TLS != nil {
		entry = entry.WithFields(logrus.Fields{
			"ssl_version": r.TLS.Version,
			"ssl_ciphers": r.TLS.CipherSuite,
		})
	}

	return entry
}

func GetApiRequestLogger(r *http.Request) *logrus.Entry {
	request_id := middleware.GetReqID(r.Context())
	apiRequestLogger := logrus.New()
	if !debug {
		apiRequestLogger.Formatter = &logrus.JSONFormatter{}
	}
	apiRequestLogger.Level = logrus.DebugLevel

	return apiRequestLogger.WithFields(logrus.Fields{"reqID": request_id})
}

func logRequest(log *logrus.Logger, logEntry *logrus.Entry, w middleware.WrapResponseWriter, dt time.Duration) {
	logEntry = logEntry.WithFields(logrus.Fields{
		"status":        w.Status(),
		"text_status":   http.StatusText(w.Status()),
		"bytes_written": w.BytesWritten(),
		"took_ns":       dt.Nanoseconds(),
	})

	logEntry.Info("Request completed")
}
