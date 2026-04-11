package admin

import "testing"

func TestNewApistationHealthHandler(t *testing.T) {
	h := NewApistationHealthHandler(nil, nil)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}
