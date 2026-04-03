package server

import (
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/urfave/negroni"
)

func logAttrsFromRequest(r *http.Request) []any {
	// Try to get the real IP
	remoteAddr := r.RemoteAddr
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		remoteAddr = realIP
	}

	return []any{
		"request", r.RequestURI,
		"method", r.Method,
		"remote", remoteAddr,
	}
}

type slogMiddleware struct {
	log          *slog.Logger
	excludePaths []string
}

func newSlogMiddleware(log *slog.Logger, excludePaths []string) *slogMiddleware {
	return &slogMiddleware{
		log:          log,
		excludePaths: excludePaths,
	}
}

func (lm *slogMiddleware) shouldLog(r *http.Request) bool {
	return !slices.Contains(lm.excludePaths, r.URL.Path)
}

func (lm *slogMiddleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if !lm.shouldLog(r) {
		next(rw, r)
		return
	}

	start := time.Now()

	entry := lm.log.With(logAttrsFromRequest(r)...)
	entry.Info("started handling request")

	next(rw, r)

	res := rw.(negroni.ResponseWriter)

	encoding := res.Header().Get("Content-Encoding")
	if encoding == "" {
		encoding = "none"
	}

	attrs := []any{
		"status", res.Status(),
		"text_status", http.StatusText(res.Status()),
		"took", time.Since(start),
		"size_byte", res.Size(),
		"encoding", encoding,
	}

	name, ok := r.Context().Value(tokenName).(string)
	if ok {
		attrs = append(attrs, "token_name", name)
	}

	entry.With(attrs...).Info("completed handling request")
}
