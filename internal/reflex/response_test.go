package reflex

import (
	"encoding/json"
	"testing"
)

func TestInterceptResponse_EchoesID(t *testing.T) {
	id := json.RawMessage(`42`)
	resp := InterceptResponse(id)
	if string(resp.ID) != string(id) {
		t.Errorf("ID = %s, want %s", resp.ID, id)
	}
}

func TestInterceptResponse_IsErrorFalse(t *testing.T) {
	resp := InterceptResponse(json.RawMessage(`1`))
	out, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.Result.IsError {
		t.Error("isError should be false")
	}
	if len(decoded.Result.Content) != 1 || decoded.Result.Content[0].Text == "" {
		t.Error("expected one non-empty text content block")
	}
}
