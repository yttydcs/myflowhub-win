// 本文件实现本地 stdio MCP 服务端，包括协议协商和 JSON-RPC 请求分发。

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

type HandleResult struct {
	Response *responseMessage
	Exit     bool
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

type toolErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
	Details any    `json:"details,omitempty"`
}

func NewServer(config ServerConfig) (*Server, error) {
	// NewServer 校验 stdio 依赖并预建工具索引，避免请求处理阶段再做结构性失败。
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
	// Serve 按行消费 stdio 上的 JSON-RPC，请求结束或收到 exit/cancel 时退出。
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
	// handleLine 解析单条请求，并把协议级错误统一翻译成 JSON-RPC 响应。
	result, err := s.HandleRequest(ctx, line)
	if err != nil {
		return err
	}
	if result.Response == nil {
		if result.Exit {
			return io.EOF
		}
		return nil
	}
	if err := s.writeResponse(*result.Response); err != nil {
		return err
	}
	if result.Exit {
		return io.EOF
	}
	return nil
}

func (s *Server) HandleRequest(ctx context.Context, data []byte) (HandleResult, error) {
	// HandleRequest 处理单条 JSON-RPC 消息，供 stdio 和 HTTP transport 共享协议语义。
	data = []byte(strings.TrimSpace(string(data)))
	if len(data) == 0 {
		return HandleResult{}, nil
	}

	var req requestMessage
	if err := json.Unmarshal(data, &req); err != nil {
		return HandleResult{Response: &responseMessage{
			JSONRPC: "2.0",
			ID:      json.RawMessage("null"),
			Error:   &responseError{Code: -32700, Message: "parse error"},
		}}, nil
	}
	if strings.TrimSpace(req.JSONRPC) == "" {
		req.JSONRPC = "2.0"
	}
	if req.JSONRPC != "2.0" {
		return HandleResult{Response: errorResponse(req.ID, -32600, "invalid request")}, nil
	}
	if strings.TrimSpace(req.Method) == "" {
		return HandleResult{Response: errorResponse(req.ID, -32600, "invalid request")}, nil
	}

	result, respond, err := s.dispatch(ctx, req)
	if err != nil {
		if !respond {
			if errors.Is(err, io.EOF) {
				return HandleResult{Exit: true}, nil
			}
			return HandleResult{}, err
		}
		var rpcErr *rpcError
		if errors.As(err, &rpcErr) {
			return HandleResult{Response: errorResponse(req.ID, rpcErr.code, rpcErr.message)}, nil
		}
		return HandleResult{Response: errorResponse(req.ID, -32603, err.Error())}, nil
	}
	if !respond {
		return HandleResult{}, nil
	}
	return HandleResult{Response: &responseMessage{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}}, nil
}

func (s *Server) dispatch(ctx context.Context, req requestMessage) (any, bool, error) {
	// dispatch 只负责 MCP 方法路由，不承担具体工具参数校验。
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
	// handleInitialize 协商双方可接受的协议版本，并返回当前 server 的静态能力描述。
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
			"tools": map[string]any{
				"listChanged": false,
			},
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
	// handleToolsCall 校验工具名与参数对象后，把调用权交给对应 ToolHandler。
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
		return errorResult(
			"unknown_tool",
			fmt.Sprintf("unknown tool: %s", name),
			"Call tools/list to inspect the available tool set.",
			map[string]any{"tool": name},
		), true, nil
	}
	if len(params.Arguments) == 0 {
		params.Arguments = json.RawMessage("{}")
	}
	return tool.Handler(ctx, params.Arguments), true, nil
}

func (s *Server) replyError(id json.RawMessage, code int, message string) error {
	return s.writeResponse(*errorResponse(id, code, message))
}

func errorResponse(id json.RawMessage, code int, message string) *responseMessage {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	return &responseMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &responseError{
			Code:    code,
			Message: strings.TrimSpace(message),
		},
	}
}

func (s *Server) writeResponse(resp responseMessage) error {
	// writeResponse 保持“一行一个 JSON-RPC 响应”，匹配当前 stdio 传输约定。
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

func errorResult(code, message, hint string, value any) CallToolResult {
	payload := toolErrorPayload{
		Code:    strings.TrimSpace(code),
		Message: strings.TrimSpace(message),
		Hint:    strings.TrimSpace(hint),
	}
	if value != nil {
		payload.Details = value
	}
	return CallToolResult{
		Content:           []toolContent{{Type: "text", Text: renderText(payload)}},
		StructuredContent: payload,
		IsError:           true,
	}
}

func renderText(value any) string {
	// renderText 把结构化结果转成可读文本，供 MCP content/text 同时展示给人和机器。
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}
