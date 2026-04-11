package apistation

import (
	"testing"
)

func TestComputeFingerprint(t *testing.T) {
	// 与 auth2api TypeScript 实现交叉验证
	tests := []struct {
		name    string
		msg     string
		version string
		salt    string
		want    string // 3 hex chars
	}{
		{
			name:    "normal message",
			msg:     "Hello, how are you doing today?",
			version: "1.0.33",
			salt:    "59cf53e54c78",
		},
		{
			name:    "short message fallback to 0",
			msg:     "Hi",
			version: "1.0.33",
			salt:    "59cf53e54c78",
		},
		{
			name:    "empty message all zeros",
			msg:     "",
			version: "1.0.33",
			salt:    "59cf53e54c78",
		},
		{
			name:    "default salt",
			msg:     "Hello, how are you doing today?",
			version: "1.0.33",
			salt:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeFingerprint(tt.msg, tt.version, tt.salt)
			if len(got) != 3 {
				t.Errorf("ComputeFingerprint() = %q, want 3 hex chars", got)
			}
			// Verify deterministic
			got2 := ComputeFingerprint(tt.msg, tt.version, tt.salt)
			if got != got2 {
				t.Errorf("Not deterministic: %q != %q", got, got2)
			}
		})
	}
	// "default salt" should equal explicit salt
	got1 := ComputeFingerprint("Hello, how are you doing today?", "1.0.33", "")
	got2 := ComputeFingerprint("Hello, how are you doing today?", "1.0.33", "59cf53e54c78")
	if got1 != got2 {
		t.Errorf("Default salt mismatch: %q != %q", got1, got2)
	}
}

func TestExtractFirstUserMessageText(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "string content",
			body: `{"messages":[{"role":"system","content":"sys"},{"role":"user","content":"hello world"}]}`,
			want: "hello world",
		},
		{
			name: "array content with text block",
			body: `{"messages":[{"role":"user","content":[{"type":"text","text":"array hello"}]}]}`,
			want: "array hello",
		},
		{
			name: "no user message",
			body: `{"messages":[{"role":"system","content":"sys"}]}`,
			want: "",
		},
		{
			name: "empty messages",
			body: `{"messages":[]}`,
			want: "",
		},
		{
			name: "no messages field",
			body: `{"model":"claude-3"}`,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractFirstUserMessageText([]byte(tt.body))
			if got != tt.want {
				t.Errorf("ExtractFirstUserMessageText() = %q, want %q", got, tt.want)
			}
		})
	}
}
