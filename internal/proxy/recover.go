package proxy

import (
	"github.com/shyam-s00/dashpot/internal/mcp"
	"github.com/shyam-s00/dashpot/internal/reflex"
)

// safeDecide calls decide, falling back to ActionPass if it panics —
// an internal bug must never block a legitimate tool call. The panic
// is logged, not swallowed, so the fallback stays visible.
func (p *Proxy) safeDecide(line []byte) (action reflex.Action, req mcp.Request) {
	defer func() {
		if r := recover(); r != nil {
			p.logf("internal error while inspecting a tool call (%v); forwarded unchanged", r)
			action, req = reflex.ActionPass, mcp.Request{}
		}
	}()
	return p.decide(line)
}
