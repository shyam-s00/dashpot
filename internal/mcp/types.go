package mcp

import "encoding/json"

// MethodToolsCall is the only method that gets inspected; every other
// method is fast pass-through.
const MethodToolsCall = "tools/call"

// Request is the subset of a JSON-RPC 2.0 request Dashpot needs. ID
// stays a json.RawMessage so it echoes back byte-for-byte regardless of
// whether it's a string, number, or null.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// ToolCallParams is Request.Params decoded, valid only when
// Request.Method == MethodToolsCall.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// IsToolCall reports whether method needs interception handling.
func IsToolCall(method string) bool {
	return method == MethodToolsCall
}

// Response is a JSON-RPC 2.0 response envelope, used only to build
// Dashpot's own synthetic replies — real tool responses stream through
// unparsed.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error is a JSON-RPC 2.0 error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
