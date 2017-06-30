package logger

import (
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/go-chi/chi/middleware"
)

const httpProtoMajor = 1

func RequestLogger(next http.Handler) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		entry := extractFromReq(r)
		lw := middleware.NewWrapResponseWriter(w, httpProtoMajor)

		t1 := time.Now()
		defer func() {
			t2 := time.Now()
			logRequest(entry, lw, t2.Sub(t1))
		}()

		next.ServeHTTP(lw, r)
	}

	return http.HandlerFunc(handler)
}

func extractFromReq(r *http.Request) *logrus.Entry {
	user := "unknown"
	reqID := middleware.GetReqID(r.Context())
	entry := logrus.WithFields(logrus.Fields{
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

func MakeAPILogger(r *http.Request) *logrus.Entry {
	reqID := middleware.GetReqID(r.Context())

	return logrus.WithFields(logrus.Fields{"reqID": reqID, "namespace": "api"})
}

func logRequest(logEntry *logrus.Entry, w middleware.WrapResponseWriter, dt time.Duration) {
	logEntry = logEntry.WithFields(logrus.Fields{
		"status":        w.Status(),
		"text_status":   http.StatusText(w.Status()),
		"bytes_written": w.BytesWritten(),
		"took_ns":       dt.Nanoseconds(),
	})

	logEntry.Debug("Request completed")
}
