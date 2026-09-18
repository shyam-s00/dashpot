package reflex

import (
	"encoding/json"

	"github.com/shyam-s00/dashpot/internal/mcp"
)

// breakerMessage avoids claiming a precise elapsed time or count: the
// sketch's decay is a continuous estimate, not a literal window, so
// the wording only says what a trip actually means.
const breakerMessage = "[DASHPOT CIRCUIT BREAKER TRIGGERED]\n\n" +
	"You have executed this exact operational pattern repeatedly in rapid succession without progress.\n\n" +
	"Execution has been paused to prevent token exhaustion and context pollution.\n" +
	"ACTION REQUIRED: Stop retrying this command. Review the outputs of your previous attempts, formulate an alternate hypothesis, and explain your new approach before calling another tool."

type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// toolCallResult is the `result` field of a synthetic tools/call reply.
// isError stays false so client frameworks treat it as normal tool
// output rather than a crash to retry.
type toolCallResult struct {
	Content []content `json:"content"`
	IsError bool      `json:"isError"`
}

// InterceptResponse builds the synthetic JSON-RPC response sent instead
// of calling the tool server, echoing id back untouched.
func InterceptResponse(id json.RawMessage) mcp.Response {
	return mcp.Response{
		JSONRPC: "2.0",
		ID:      id,
		Result: toolCallResult{
			Content: []content{{Type: "text", Text: breakerMessage}},
			IsError: false,
		},
	}
}
