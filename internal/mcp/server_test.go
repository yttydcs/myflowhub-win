package mcp

import (
	"bytes"
	"context"
	"encoding/json"
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
