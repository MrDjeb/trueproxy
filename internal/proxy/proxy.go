package proxy

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/mrdjeb/trueproxy/internal/logger/sl"
	"github.com/mrdjeb/trueproxy/internal/storage"
)

const (
	HTTPS = "https"
	HTTP  = "http"
)

type ProxyHandler struct {
	log         *slog.Logger
	TransortTLS *http.Transport
	cm          *CertManager
	rt          http.RoundTripper
}

func New(log *slog.Logger, cm *CertManager, repo storage.RequestsRepo) *ProxyHandler {
	rt := &proxyRoundTripper{
		next: http.DefaultTransport,
		log:  log,
		repo: repo,
	}
	return &ProxyHandler{
		log: log,
		cm:  cm,
		rt:  rt,
	}

}

func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleHTTPS(w, r)
		return
	}
	p.handleHTTP(w, r)
}

func (p *ProxyHandler) handleHTTP(respW http.ResponseWriter, inReq *http.Request) {
	requestID := uuid.New().String()
	proto := HTTP

	log := p.log.With(
		slog.String("Proto", proto),
		slog.String("request-ID", requestID),
	)

	//= = = = = = = Hijack client = = = = = = =//
	rc := http.NewResponseController(respW)
	rc.EnableFullDuplex()

	cleanClientConn, _, err := rc.Hijack()
	if err != nil {
		log.Error("hijacking fail", sl.Err(err))
		writeRawClientResponse(log, inReq, cleanClientConn, http.StatusInternalServerError)
		return
	}
	defer cleanClientConn.Close()

	//- - - - - - - Hijack client - - - - - - -//

	responseDump, err := p.handleSingle(inReq, proto)
	if err != nil {
		log.Error("handle single error", sl.Err(err))
		writeRawClientResponse(log, inReq, cleanClientConn, http.StatusBadGateway)
		return
	}

	if _, err := cleanClientConn.Write(responseDump); err != nil {
		log.Error("error writing response back to client connection", sl.Err(err))
		return
	}
}

func (p *ProxyHandler) handleHTTPS(respW http.ResponseWriter, inReq *http.Request) {
	requestID := uuid.New().String()
	proto := HTTPS

	log := p.log.With(
		slog.String("Proto", proto),
		slog.String("request-ID", requestID),
	)

	//= = = = = = = Hijack client = = = = = = =//
	rc := http.NewResponseController(respW)
	rc.EnableFullDuplex()

	cleanClientConn, _, err := rc.Hijack()
	if err != nil {
		log.Error("hijacking fail", sl.Err(err))
		writeRawClientResponse(log, inReq, cleanClientConn, http.StatusInternalServerError)
		return
	}
	defer cleanClientConn.Close()
	//- - - - - - - Hijack client - - - - - - -//

	//= = = = = = = Setup TLS = = = = = = =//
	if _, err := cleanClientConn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		log.Error("error accept CONNECT_METHOD to client:", sl.Err(err))
		return
	}

	tlsClientConn := tls.Server(cleanClientConn, p.cm.NewTLSConfig(inReq.URL.Host))
	defer tlsClientConn.Close()

	connReader := bufio.NewReader(tlsClientConn)

	r, err := http.ReadRequest(connReader) // only supports HTTP/1.x requests
	opErr, okOp := err.(*net.OpError)
	netErr, okNet := err.(net.Error)
	switch {
	case err == io.EOF:
		log.Info("Client close the connection")
		return
	case errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNABORTED):
		log.Error("Client connection reset by peer error", sl.Err(err))
		return
	case okNet:
		if !netErr.Timeout() {
			log.Warn("Client connection force close, tcp", sl.Err(netErr))
			return
		} else {
			log.Error("Read request from client connection error, netERr", sl.Err(netErr))
			return
		}
	case okOp:
		if opErr.Op == "read" {
			log.Info("Client read error, tcp")
			return
		} else {
			log.Error("Read request from client connection error, opErr", sl.Err(opErr))
			return
		}
	case err != nil:
		log.Error("Read request from client connection error", sl.Err(err))
		return
	}

	//- - - - - - - Setup TLS - - - - - - -//

	responseDump, err := p.handleSingle(r, proto)
	if err != nil {
		log.Error("handle single error", sl.Err(err))
		writeRawClientResponse(log, inReq, cleanClientConn, http.StatusBadGateway)
		return
	}

	if _, err := tlsClientConn.Write(responseDump); err != nil {
		log.Error("error writing response back to client connection", sl.Err(err))
		return
	}
}

func (p *ProxyHandler) handleSingle(inReq *http.Request, proto string) ([]byte, error) {

	ctx := inReq.Context()
	outReq := inReq.Clone(ctx)

	// for https only
	changeRequestToTarget(outReq, inReq.Host, proto)

	outReq.RequestURI = ""
	if inReq.ContentLength == 0 {
		outReq.Body = nil
	}
	if outReq.Body != nil {
		defer outReq.Body.Close()
	}
	if outReq.Header == nil {
		outReq.Header = make(http.Header)
	}
	outReq.Close = false

	removeHopByHopHeaders(outReq.Header)

	if _, ok := outReq.Header["User-Agent"]; !ok {
		outReq.Header.Set("User-Agent", "")
	}

	client := http.Client{
		Transport: p.rt,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(outReq)
	if err != nil {
		return nil, fmt.Errorf("error in client DO: %w", err)
	}
	defer resp.Body.Close()

	removeHopByHopHeaders(resp.Header)

	return httputil.DumpResponse(resp, true)
}

func changeRequestToTarget(req *http.Request, targetHost string, proto string) error {
	if proto != HTTPS {
		return nil
	}

	if !strings.HasPrefix(targetHost, "https") {
		targetHost = "https://" + targetHost
	}
	targetUrl, err := url.Parse(targetHost)
	if err != nil {
		return err
	}

	targetUrl.Path = req.URL.Path
	targetUrl.RawQuery = req.URL.RawQuery
	req.URL = targetUrl

	req.RequestURI = ""
	return nil
}

func writeRawClientResponse(log *slog.Logger, r *http.Request, tlsClientConn net.Conn, statusCode int) bool {
	if _, err := fmt.Fprintf(tlsClientConn, "HTTP/%d.%d %03d %s\r\n\r\n",
		r.ProtoMajor, r.ProtoMinor,
		statusCode, http.StatusText(statusCode)); err != nil {
		log.Error("writing response failed", "addr", tlsClientConn.RemoteAddr(), sl.Err(err))
		return false
	}
	return true
}

var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

func removeHopByHopHeaders(h http.Header) {
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = textproto.TrimString(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
	for _, f := range hopHeaders {
		h.Del(f)
	}
}
