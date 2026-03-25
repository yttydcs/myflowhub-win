package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

const latestProtocolVersion = "2025-11-25"

var supportedProtocolVersions = map[string]struct{}{
	"2024-11-05": {},
	"2025-03-26": {},
	"2025-11-25": {},
}

type ServerConfig struct {
	Name         string
	Version      string
	Instructions string
	Reader       io.Reader
	Writer       io.Writer
	Tools        []Tool
}

type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	Handler     ToolHandler
}

type ToolHandler func(ctx context.Context, arguments json.RawMessage) CallToolResult

type Server struct {
	name         string
	version      string
	instructions string
	reader       *bufio.Reader
	writer       io.Writer

	mu       sync.Mutex
	tools    []Tool
	toolByID map[string]Tool
}

type requestMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type responseMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *responseError  `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcError struct {
	code    int
	message string
}

func (e *rpcError) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

type initializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      serverInfo     `json:"serverInfo"`
	Instructions    string         `json:"instructions,omitempty"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type toolsListResult struct {
	Tools []toolDescriptor `json:"tools"`
}

type toolDescriptor struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema"`
}

type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content           []toolContent `json:"content,omitempty"`
	StructuredContent any           `json:"structuredContent,omitempty"`
	IsError           bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func NewServer(config ServerConfig) (*Server, error) {
	if config.Reader == nil {
		return nil, errors.New("reader is required")
	}
	if config.Writer == nil {
		return nil, errors.New("writer is required")
	}
	name := strings.TrimSpace(config.Name)
	if name == "" {
		name = "myflowhub-mcp"
	}
	version := strings.TrimSpace(config.Version)
	if version == "" {
		version = "dev"
	}
	srv := &Server{
		name:         name,
		version:      version,
		instructions: strings.TrimSpace(config.Instructions),
		reader:       bufio.NewReader(config.Reader),
		writer:       config.Writer,
		tools:        make([]Tool, 0, len(config.Tools)),
		toolByID:     make(map[string]Tool, len(config.Tools)),
	}
	for _, tool := range config.Tools {
		if strings.TrimSpace(tool.Name) == "" {
			return nil, errors.New("tool name is required")
		}
		srv.tools = append(srv.tools, tool)
		srv.toolByID[tool.Name] = tool
	}
	return srv, nil
}

func (s *Server) Serve(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := s.reader.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) && len(strings.TrimSpace(string(line))) == 0 {
				return nil
			}
			if errors.Is(err, io.EOF) {
				if handleErr := s.handleLine(ctx, line); handleErr != nil {
					return handleErr
				}
				return nil
			}
			return err
		}
		if err := s.handleLine(ctx, line); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func (s *Server) handleLine(ctx context.Context, line []byte) error {
	line = []byte(strings.TrimSpace(string(line)))
	if len(line) == 0 {
		return nil
	}

	var req requestMessage
	if err := json.Unmarshal(line, &req); err != nil {
		return s.writeResponse(responseMessage{
			JSONRPC: "2.0",
			ID:      json.RawMessage("null"),
			Error:   &responseError{Code: -32700, Message: "parse error"},
		})
	}
	if strings.TrimSpace(req.JSONRPC) == "" {
		req.JSONRPC = "2.0"
	}
	if req.JSONRPC != "2.0" {
		return s.replyError(req.ID, -32600, "invalid request")
	}
	if strings.TrimSpace(req.Method) == "" {
		return s.replyError(req.ID, -32600, "invalid request")
	}

	result, respond, err := s.dispatch(ctx, req)
	if err != nil {
		if !respond {
			return err
		}
		var rpcErr *rpcError
		if errors.As(err, &rpcErr) {
			return s.replyError(req.ID, rpcErr.code, rpcErr.message)
		}
		return s.replyError(req.ID, -32603, err.Error())
	}
	if !respond {
		return nil
	}
	return s.writeResponse(responseMessage{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	})
}

func (s *Server) dispatch(ctx context.Context, req requestMessage) (any, bool, error) {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.Params)
	case "ping", "shutdown":
		return map[string]any{}, true, nil
	case "notifications/initialized", "notifications/cancelled":
		return nil, false, nil
	case "exit":
		return nil, false, io.EOF
	case "tools/list":
		return s.handleToolsList(), true, nil
	case "tools/call":
		return s.handleToolsCall(ctx, req.Params)
	default:
		return nil, true, &rpcError{code: -32601, message: fmt.Sprintf("method not found: %s", req.Method)}
	}
}

func (s *Server) handleInitialize(raw json.RawMessage) (initializeResult, bool, error) {
	params := initializeParams{}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &params); err != nil {
			return initializeResult{}, true, fmt.Errorf("invalid initialize params: %w", err)
		}
	}
	version := latestProtocolVersion
	if _, ok := supportedProtocolVersions[strings.TrimSpace(params.ProtocolVersion)]; ok {
		version = strings.TrimSpace(params.ProtocolVersion)
	}
	return initializeResult{
		ProtocolVersion: version,
		Capabilities: map[string]any{
			"tools": map[string]any{},
		},
		ServerInfo: serverInfo{
			Name:    s.name,
			Version: s.version,
		},
		Instructions: s.instructions,
	}, true, nil
}

func (s *Server) handleToolsList() toolsListResult {
	tools := make([]toolDescriptor, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, toolDescriptor{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}
	return toolsListResult{Tools: tools}
}

func (s *Server) handleToolsCall(ctx context.Context, raw json.RawMessage) (CallToolResult, bool, error) {
	params := toolsCallParams{}
	if len(raw) == 0 || string(raw) == "null" {
		return CallToolResult{}, true, &rpcError{code: -32602, message: "invalid tools/call params"}
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return CallToolResult{}, true, &rpcError{code: -32602, message: fmt.Sprintf("invalid tools/call params: %v", err)}
	}
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return CallToolResult{}, true, &rpcError{code: -32602, message: "tool name is required"}
	}
	tool, ok := s.toolByID[name]
	if !ok {
		return errorResult(fmt.Sprintf("unknown tool: %s", name), map[string]any{"tool": name}), true, nil
	}
	if len(params.Arguments) == 0 {
		params.Arguments = json.RawMessage("{}")
	}
	return tool.Handler(ctx, params.Arguments), true, nil
}

func (s *Server) replyError(id json.RawMessage, code int, message string) error {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	return s.writeResponse(responseMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &responseError{
			Code:    code,
			Message: strings.TrimSpace(message),
		},
	})
}

func (s *Server) writeResponse(resp responseMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := s.writer.Write(data); err != nil {
		return err
	}
	_, err = s.writer.Write([]byte("\n"))
	return err
}

func successResult(value any) CallToolResult {
	return CallToolResult{
		Content:           []toolContent{{Type: "text", Text: renderText(value)}},
		StructuredContent: value,
	}
}

func errorResult(message string, value any) CallToolResult {
	payload := map[string]any{
		"error": strings.TrimSpace(message),
	}
	if value != nil {
		payload["details"] = value
	}
	return CallToolResult{
		Content:           []toolContent{{Type: "text", Text: renderText(payload)}},
		StructuredContent: payload,
		IsError:           true,
	}
}

func renderText(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}
