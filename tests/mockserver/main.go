// Command mockserver is a minimal JSON-RPC 2.0 stdio MCP tool server
// exposing one "bash" tool, for exercising a real dashpot binary
// end-to-end without needing a real MCP implementation.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type toolCallParams struct {
	Name string `json:"name"`
}

func main() {
	fail := flag.Bool("fail", false, "return an error result for every bash call")
	flag.Parse()

	count := 0
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	for scanner.Scan() {
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}

		switch req.Method {
		case "initialize":
			writeResult(out, req.ID, map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{},
			})
		case "tools/list":
			writeResult(out, req.ID, map[string]any{
				"tools": []map[string]any{{"name": "bash"}},
			})
		case "tools/call":
			var params toolCallParams
			_ = json.Unmarshal(req.Params, &params)
			if params.Name == "bash" {
				count++
			}
			writeResult(out, req.ID, toolResult(*fail, count))
		case "test/invocationCount":
			writeResult(out, req.ID, map[string]any{"count": count})
		default:
			writeResult(out, req.ID, map[string]any{})
		}
		out.Flush()
	}
}

func toolResult(fail bool, count int) map[string]any {
	if fail {
		return map[string]any{
			"content": []map[string]any{{"type": "text", "text": "mock failure"}},
			"isError": true,
		}
	}
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": fmt.Sprintf("ran (invocation %d)", count)}},
		"isError": false,
	}
}

func writeResult(w *bufio.Writer, id json.RawMessage, result any) {
	b, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
	if err != nil {
		return
	}
	w.Write(b)
	w.WriteByte('\n')
}
