package reflex

// Exemptions holds tool names that bypass the reflex engine, and
// per-tool threshold overrides — both explicit, user-configured maps.
type Exemptions struct {
	skip      map[string]struct{}
	threshold map[string]uint32
}

// NewExemptions builds an Exemptions set from a skip list and a
// tool-name-to-threshold override map.
func NewExemptions(skip []string, thresholds map[string]uint32) *Exemptions {
	skipSet := make(map[string]struct{}, len(skip))
	for _, name := range skip {
		skipSet[name] = struct{}{}
	}
	return &Exemptions{skip: skipSet, threshold: thresholds}
}

// Skip reports whether tool should bypass Decide entirely.
func (ex *Exemptions) Skip(tool string) bool {
	if ex == nil {
		return false
	}
	_, ok := ex.skip[tool]
	return ok
}

// Threshold returns tool's overridden threshold and true, or
// (0, false) if tool has no override.
func (ex *Exemptions) Threshold(tool string) (uint32, bool) {
	if ex == nil {
		return 0, false
	}
	t, ok := ex.threshold[tool]
	return t, ok
}
