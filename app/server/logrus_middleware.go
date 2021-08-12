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
	logger *logrus.Logger
}

func newLogrusMiddleware(logger *logrus.Logger) *logrusMiddleware {
	return &logrusMiddleware{
		logger: logger,
	}
}

func (lm *logrusMiddleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
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

type logrusAuthMiddleware struct {
	logger *logrus.Logger
}

func newLogrusAuthMiddleware(logger *logrus.Logger) *logrusAuthMiddleware {
	return &logrusAuthMiddleware{
		logger: logger,
	}
}

func (lam *logrusAuthMiddleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	tokenName, ok := r.Context().Value(auth.TokenName).(string)
	if ok {
		logrus.
			NewEntry(lam.logger).
			WithFields(logFieldsFromRequest(r)).
			WithField("token_name", tokenName).
			Info("authorized")
	}

	next(rw, r)
}
