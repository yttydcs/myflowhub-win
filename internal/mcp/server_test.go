// 本文件覆盖本地 MCP 服务端的协议协商与 JSON-RPC 请求处理行为。

package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestServerInitializeAndToolsList(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","method":"exit"}`,
		"",
	}, "\n")

	var output bytes.Buffer
	server, err := NewServer(ServerConfig{
		Name:    "myflowhub-mcp",
		Version: "test",
		Reader:  strings.NewReader(input),
		Writer:  &output,
		Tools: []Tool{
			{
				Name:        "demo_tool",
				Description: "demo",
				InputSchema: objectSchema(nil),
				Handler: func(context.Context, json.RawMessage) CallToolResult {
					return successResult(map[string]any{"ok": true})
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if err := server.Serve(context.Background()); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 response lines, got %d: %q", len(lines), output.String())
	}

	var initResp map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("unmarshal initialize response: %v", err)
	}
	result, ok := initResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing initialize result: %#v", initResp)
	}
	if got := result["protocolVersion"]; got != latestProtocolVersion {
		t.Fatalf("protocolVersion = %v", got)
	}
	capabilities, ok := result["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("missing capabilities: %#v", result)
	}
	toolsCapability, ok := capabilities["tools"].(map[string]any)
	if !ok || toolsCapability["listChanged"] != false {
		t.Fatalf("unexpected tools capability: %#v", capabilities["tools"])
	}

	var listResp map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &listResp); err != nil {
		t.Fatalf("unmarshal tools/list response: %v", err)
	}
	listResult, ok := listResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing list result: %#v", listResp)
	}
	tools, ok := listResult["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("unexpected tools payload: %#v", listResult["tools"])
	}
}

func TestServerUnknownToolReturnsStructuredToolError(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"missing_tool","arguments":{}}}`,
		`{"jsonrpc":"2.0","method":"exit"}`,
		"",
	}, "\n")

	var output bytes.Buffer
	server, err := NewServer(ServerConfig{
		Name:    "myflowhub-mcp",
		Version: "test",
		Reader:  strings.NewReader(input),
		Writer:  &output,
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if err := server.Serve(context.Background()); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 response line, got %d: %q", len(lines), output.String())
	}

	var resp struct {
		Result CallToolResult `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &resp); err != nil {
		t.Fatalf("unmarshal tools/call response: %v", err)
	}
	if !resp.Result.IsError {
		t.Fatalf("expected tool error, got %#v", resp.Result)
	}
	payload, ok := resp.Result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structured content map, got %#v", resp.Result.StructuredContent)
	}
	if payload["code"] != "unknown_tool" {
		t.Fatalf("unexpected error payload: %#v", payload)
	}
}

func TestServerHandleRequestCanReuseDispatchWithoutStdio(t *testing.T) {
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
				Handler: func(context.Context, json.RawMessage) CallToolResult {
					return successResult(map[string]any{"called": true})
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	result, err := server.HandleRequest(context.Background(), []byte(`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"demo_tool","arguments":{}}}`))
	if err != nil {
		t.Fatalf("HandleRequest() error = %v", err)
	}
	if result.Exit {
		t.Fatal("HandleRequest() unexpectedly requested exit")
	}
	if result.Response == nil {
		t.Fatal("HandleRequest() missing response")
	}
	if string(result.Response.ID) != "7" {
		t.Fatalf("response id = %s", result.Response.ID)
	}
	callResult, ok := result.Response.Result.(CallToolResult)
	if !ok {
		t.Fatalf("response result type = %T", result.Response.Result)
	}
	if callResult.IsError {
		t.Fatalf("expected success result, got %#v", callResult)
	}
}
