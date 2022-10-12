package server

import (
	"net/http"
	"time"

	"github.com/odwrtw/polochon/app/auth"
	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"
)

func logFieldsFromRequest(r *http.Request) logrus.Fields {
	// Try to get the real IP
	remoteAddr := r.RemoteAddr
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		remoteAddr = realIP
	}

	return logrus.Fields{
		"request": r.RequestURI,
		"method":  r.Method,
		"remote":  remoteAddr,
	}
}

type logrusMiddleware struct {
	logger       *logrus.Logger
	excludePaths []string
}

func newLogrusMiddleware(logger *logrus.Logger, excludePaths []string) *logrusMiddleware {
	return &logrusMiddleware{
		logger:       logger,
		excludePaths: excludePaths,
	}
}

func (lm *logrusMiddleware) shouldLog(r *http.Request) bool {
	for _, excluded := range lm.excludePaths {
		if r.URL.Path == excluded {
			return false
		}
	}

	return true
}

func (lm *logrusMiddleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if !lm.shouldLog(r) {
		next(rw, r)
		return
	}

	start := time.Now()

	entry := logrus.NewEntry(lm.logger)
	entry = entry.WithFields(logFieldsFromRequest(r))

	entry.Info("started handling request")

	next(rw, r)

	res := rw.(negroni.ResponseWriter)
	entry = entry.WithFields(logrus.Fields{
		"status":      res.Status(),
		"text_status": http.StatusText(res.Status()),
		"took":        time.Since(start),
		"size_byte":   res.Size(),
	})

	tokenName, ok := r.Context().Value(auth.TokenName).(string)
	if ok {
		entry = entry.WithField("token_name", tokenName)
	}

	entry.Info("completed handling request")
}
