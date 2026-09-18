package reflex

import "testing"

func TestExemptions_Skip(t *testing.T) {
	ex := NewExemptions([]string{"get_status", "poll_job"}, nil)

	if !ex.Skip("get_status") {
		t.Error("get_status should be skipped")
	}
	if ex.Skip("bash") {
		t.Error("bash should not be skipped")
	}
}

func TestExemptions_Threshold(t *testing.T) {
	ex := NewExemptions(nil, map[string]uint32{"get_status": 50})

	if got, ok := ex.Threshold("get_status"); !ok || got != 50 {
		t.Errorf("Threshold(get_status) = %d, %v; want 50, true", got, ok)
	}
	if _, ok := ex.Threshold("bash"); ok {
		t.Error("bash should have no threshold override")
	}
}

func TestExemptions_NilIsSafe(t *testing.T) {
	var ex *Exemptions
	if ex.Skip("bash") {
		t.Error("nil Exemptions should never skip")
	}
	if _, ok := ex.Threshold("bash"); ok {
		t.Error("nil Exemptions should never override a threshold")
	}
}
