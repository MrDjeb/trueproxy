package proxy

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
)

type ProxyHandler struct {
	log     *slog.Logger
	counter int64
}

func New(log *slog.Logger) *ProxyHandler {
	return &ProxyHandler{
		log: log,
	}
}

func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if b, err := httputil.DumpRequest(r, true); err == nil {
		p.log.Info("handle", "Request", string(b))
	}
	p.counter++
	fmt.Fprintln(w, "Counter:", p.counter)
}
