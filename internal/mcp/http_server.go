package mcp

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultHTTPListen = "127.0.0.1:17688"
	defaultHTTPPath   = "/mcp"
)

type HTTPServerConfig struct {
	Server      *Server
	ListenAddr  string
	Path        string
	AllowRemote bool
	AuthToken   string
}

type HTTPServer struct {
	server      *Server
	listenAddr  string
	path        string
	allowRemote bool
	authToken   string
	httpServer  *http.Server
}

func NewHTTPServer(config HTTPServerConfig) (*HTTPServer, error) {
	if config.Server == nil {
		return nil, errors.New("server is required")
	}
	listenAddr := strings.TrimSpace(config.ListenAddr)
	if listenAddr == "" {
		listenAddr = defaultHTTPListen
	}
	if !config.AllowRemote && !isLoopbackListenAddr(listenAddr) {
		return nil, fmt.Errorf("listen address %q is not loopback; use an explicit unsafe remote mode before exposing MCP", listenAddr)
	}
	authToken := strings.TrimSpace(config.AuthToken)
	if config.AllowRemote && authToken == "" {
		return nil, errors.New("auth token is required when remote MCP HTTP mode is enabled")
	}
	path := normalizeHTTPPath(config.Path)
	srv := &HTTPServer{
		server:      config.Server,
		listenAddr:  listenAddr,
		path:        path,
		allowRemote: config.AllowRemote,
		authToken:   authToken,
	}
	mux := http.NewServeMux()
	mux.HandleFunc(path, srv.handleMCP)
	srv.httpServer = &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: defaultReadHeaderTimeout(),
	}
	return srv, nil
}

func (s *HTTPServer) ListenAddr() string {
	if s == nil {
		return ""
	}
	return s.listenAddr
}

func (s *HTTPServer) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *HTTPServer) Serve(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return errors.New("http server is not initialized")
	}
	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout())
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		if err := <-errCh; err != nil {
			return err
		}
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (s *HTTPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	if s.authToken != "" && !isAuthorizedBearer(r.Header.Get("Authorization"), s.authToken) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !s.allowRemote && !isAllowedOrigin(r.Header.Get("Origin")) {
		http.Error(w, "origin is not allowed", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 4*1024*1024))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		http.Error(w, "request body is required", http.StatusBadRequest)
		return
	}
	result, err := s.server.HandleRequest(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if result.Response == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	data, err := json.Marshal(result.Response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func normalizeHTTPPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return defaultHTTPPath
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func isAllowedOrigin(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return isLoopbackHost(parsed.Hostname())
}

func isAuthorizedBearer(header string, token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return true
	}
	header = strings.TrimSpace(header)
	scheme, value, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return false
	}
	value = strings.TrimSpace(value)
	if len(value) != len(token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(value), []byte(token)) == 1
}

func isLoopbackListenAddr(addr string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		host = strings.TrimSpace(addr)
	}
	if strings.TrimSpace(host) == "" {
		return false
	}
	return isLoopbackHost(host)
}

func isLoopbackHost(host string) bool {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func defaultReadHeaderTimeout() time.Duration {
	return 5 * time.Second
}

func defaultShutdownTimeout() time.Duration {
	return 5 * time.Second
}
