package reflex

// Action is the decision Decide returns for one tool call.
type Action int

const (
	ActionPass Action = iota
	ActionIntercept
)

// Decide records key and reports pass/intercept for tool. An exempt
// tool always passes without touching the sketch.
func (e *Engine) Decide(tool string, key []byte) Action {
	if e.exempt.Skip(tool) {
		return ActionPass
	}

	threshold := e.threshold
	if t, ok := e.exempt.Threshold(tool); ok {
		threshold = t
	}

	estimate, _ := e.sk.ObserveBytes(key)
	if estimate >= threshold {
		return ActionIntercept
	}
	return ActionPass
}
