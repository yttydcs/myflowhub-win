package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestHTTPServerPostToolsList(t *testing.T) {
	server := newTestHTTPMCPServer(t)
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	result, ok := payload["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %#v", payload)
	}
	tools, ok := result["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("unexpected tools: %#v", result["tools"])
	}
}

func TestHTTPServerNotificationReturnsAccepted(t *testing.T) {
	server := newTestHTTPMCPServer(t)
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`))
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", rec.Body.String())
	}
}

func TestHTTPServerRejectsGet(t *testing.T) {
	server := newTestHTTPMCPServer(t)
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("Allow = %q", got)
	}
}

func TestHTTPServerRejectsRemoteOrigin(t *testing.T) {
	server := newTestHTTPMCPServer(t)
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHTTPServerAllowsLocalhostOrigin(t *testing.T) {
	server := newTestHTTPMCPServer(t)
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	req.Header.Set("Origin", "http://127.0.0.1:17688")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestNewHTTPServerRejectsNonLoopbackByDefault(t *testing.T) {
	stdio := newTestServer(t, nil)
	for _, addr := range []string{"0.0.0.0:17688", ":17688"} {
		_, err := NewHTTPServer(HTTPServerConfig{
			Server:     stdio,
			ListenAddr: addr,
		})
		if err == nil {
			t.Fatalf("expected listen address %q to be rejected", addr)
		}
	}
}

func TestHTTPServerReusesSameToolHandlerAcrossRequests(t *testing.T) {
	var calls int32
	stdio := newTestServer(t, func(context.Context, json.RawMessage) CallToolResult {
		next := atomic.AddInt32(&calls, 1)
		return successResult(map[string]any{"calls": next})
	})
	server, err := NewHTTPServer(HTTPServerConfig{Server: stdio})
	if err != nil {
		t.Fatalf("NewHTTPServer() error = %v", err)
	}

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"demo_tool","arguments":{}}}`))
		rec := httptest.NewRecorder()
		server.handleMCP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d status = %d body=%s", i, rec.Code, rec.Body.String())
		}
	}

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("handler calls = %d", got)
	}
}

func newTestHTTPMCPServer(t *testing.T) *HTTPServer {
	t.Helper()
	stdio := newTestServer(t, nil)
	server, err := NewHTTPServer(HTTPServerConfig{Server: stdio})
	if err != nil {
		t.Fatalf("NewHTTPServer() error = %v", err)
	}
	return server
}

func newTestServer(t *testing.T, handler ToolHandler) *Server {
	t.Helper()
	if handler == nil {
		handler = func(context.Context, json.RawMessage) CallToolResult {
			return successResult(map[string]any{"ok": true})
		}
	}
	server, err := NewServer(ServerConfig{
		Name:    "myflowhub-mcp",
		Version: "test",
		Reader:  strings.NewReader(""),
		Writer:  io.Discard,
		Tools: []Tool{
			{
				Name:        "demo_tool",
				Description: "demo",
				InputSchema: objectSchema(nil),
				Handler:     handler,
			},
		},
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return server
}
