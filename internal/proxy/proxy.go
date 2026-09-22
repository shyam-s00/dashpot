// Package proxy decides, per JSON-RPC line from a client, whether to
// forward a tool call to the child server or intercept it.
package proxy

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/shyam-s00/dashpot/internal/mcp"
	"github.com/shyam-s00/dashpot/internal/normalize"
	"github.com/shyam-s00/dashpot/internal/reflex"
)

// Proxy reads JSON-RPC lines from in, forwards allowed calls to
// childIn verbatim, and writes synthetic intercept replies to out.
type Proxy struct {
	engine  *reflex.Engine
	scanner *mcp.Scanner
	childIn io.Writer
	out     io.Writer
	keys    *normalize.KeyBuilder
	logw    io.Writer
}

// New builds a Proxy. Anything Dashpot fails to inspect (a batch
// request, an oversized line, a recovered internal error) is still
// forwarded, but is first logged to logw so the bypass is never silent.
func New(engine *reflex.Engine, in io.Reader, childIn, out, logw io.Writer) *Proxy {
	return &Proxy{
		engine:  engine,
		scanner: mcp.NewScanner(in),
		childIn: childIn,
		out:     out,
		keys:    normalize.NewKeyBuilder(),
		logw:    logw,
	}
}

// logf writes a "dashpot: "-prefixed diagnostic line to logw. logw may
// be nil (e.g. in tests that don't care), in which case it's a no-op.
func (p *Proxy) logf(format string, args ...any) {
	if p.logw == nil {
		return
	}
	fmt.Fprintf(p.logw, "dashpot: "+format+"\n", args...)
}

// Run processes client requests until the stream ends or a write fails.
func (p *Proxy) Run() error {
	for {
		line, err := p.scanner.Scan(p.childIn)
		switch err {
		case io.EOF:
			return nil
		case mcp.ErrLineTooLong:
			p.logf("a request line exceeded %d bytes; forwarded unchanged without inspection", mcp.MaxLineSize)
			continue // already forwarded raw by the scanner
		case nil:
			if err := p.handleLine(line); err != nil {
				return err
			}
		default:
			return err
		}
	}
}

func (p *Proxy) handleLine(line []byte) error {
	action, req := p.safeDecide(line)
	if action == reflex.ActionPass {
		return p.forward(line)
	}
	return p.intercept(req.ID)
}

func (p *Proxy) forward(line []byte) error {
	if _, err := p.childIn.Write(line); err != nil {
		return err
	}
	_, err := p.childIn.Write([]byte("\n"))
	return err
}

func (p *Proxy) intercept(id json.RawMessage) error {
	out, err := json.Marshal(reflex.InterceptResponse(id))
	if err != nil {
		return err
	}
	if _, err := p.out.Write(out); err != nil {
		return err
	}
	_, err = p.out.Write([]byte("\n"))
	return err
}

// decide parses line as a JSON-RPC request and, for a well-formed
// tools/call, runs it through the reflex engine. Anything else passes.
func (p *Proxy) decide(line []byte) (reflex.Action, mcp.Request) {
	if isBatch(line) {
		p.logf("received a JSON-RPC batch request; forwarded unchanged, its calls were not inspected")
		return reflex.ActionPass, mcp.Request{}
	}

	var req mcp.Request
	if err := json.Unmarshal(line, &req); err != nil {
		return reflex.ActionPass, req
	}
	if !mcp.IsToolCall(req.Method) {
		return reflex.ActionPass, req
	}
	var params mcp.ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return reflex.ActionPass, req
	}

	canonical := normalize.Canonicalize(params.Arguments)
	key := p.keys.Build(params.Name, canonical)
	return p.engine.Decide(params.Name, key), req
}

// isBatch reports whether line is a JSON-RPC batch (a top-level JSON
// array) rather than a single request object.
func isBatch(line []byte) bool {
	for _, b := range line {
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		case '[':
			return true
		default:
			return false
		}
	}
	return false
}
