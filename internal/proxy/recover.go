package proxy

import (
	"github.com/shyam-s00/dashpot/internal/mcp"
	"github.com/shyam-s00/dashpot/internal/reflex"
)

// safeDecide calls decide, falling back to ActionPass if it panics —
// an internal bug must never block a legitimate tool call.
func (p *Proxy) safeDecide(line []byte) (action reflex.Action, req mcp.Request) {
	defer func() {
		if recover() != nil {
			action, req = reflex.ActionPass, mcp.Request{}
		}
	}()
	return p.decide(line)
}
