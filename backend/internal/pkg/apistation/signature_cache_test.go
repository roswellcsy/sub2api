package apistation

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestSignatureCache_PutAndGet(t *testing.T) {
	cache := &SignatureCache{}
	model := "claude-sonnet-4-6"
	text := "This is a thinking block with substantial content for testing purposes."
	sig := "ErwSig_" + strings.Repeat("a", 60)

	if got := cache.Get(model, text); got != "" {
		t.Errorf("expected empty, got %q", got)
	}

	cache.Put(model, text, sig)
	if got := cache.Get(model, text); got != sig {
		t.Errorf("expected %q, got %q", sig, got)
	}
}

func TestSignatureCache_ShortSignatureIgnored(t *testing.T) {
	cache := &SignatureCache{}
	cache.Put("claude", "text", "short")
	if got := cache.Get("claude", "text"); got != "" {
		t.Errorf("expected empty for short signature, got %q", got)
	}
}

func TestSignatureCache_EmptyText(t *testing.T) {
	cache := &SignatureCache{}
	longSig := strings.Repeat("a", 60)
	cache.Put("claude", "", longSig)
	if got := cache.Get("claude", ""); got != "" {
		t.Errorf("expected empty for empty text, got %q", got)
	}
}

func TestProcessThinkingSignatures_NoMessages(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4-6","max_tokens":1024}`)
	result := ProcessThinkingSignatures("claude-sonnet-4-6", body)
	if string(result) != string(body) {
		t.Errorf("expected unchanged body")
	}
}

func TestProcessThinkingSignatures_CachesValidSignature(t *testing.T) {
	DefaultSignatureCache = &SignatureCache{}

	sig := "ErwSig_" + strings.Repeat("a", 60)
	body := []byte(`{
		"model": "claude-sonnet-4-6",
		"messages": [
			{"role": "assistant", "content": [
				{"type": "thinking", "thinking": "test thinking content here", "signature": "` + sig + `"}
			]},
			{"role": "user", "content": "hello"}
		]
	}`)

	ProcessThinkingSignatures("claude-sonnet-4-6", body)

	got := DefaultSignatureCache.Get("claude-sonnet-4-6", "test thinking content here")
	if got != sig {
		t.Errorf("expected cached signature %q, got %q", sig, got)
	}
}

func TestProcessThinkingSignatures_InjectsCachedSignature(t *testing.T) {
	DefaultSignatureCache = &SignatureCache{}
	sig := "ErwSig_valid_" + strings.Repeat("b", 50)
	DefaultSignatureCache.Put("claude-sonnet-4-6", "thinking text to cache", sig)

	body := []byte(`{
		"model": "claude-sonnet-4-6",
		"messages": [
			{"role": "assistant", "content": [
				{"type": "thinking", "thinking": "thinking text to cache", "signature": ""}
			]},
			{"role": "user", "content": "hello"}
		]
	}`)

	result := ProcessThinkingSignatures("claude-sonnet-4-6", body)

	if got := gjson.GetBytes(result, "messages.0.content.0.signature").String(); got != sig {
		t.Errorf("expected injected signature %q, got %q", sig, got)
	}
}

func TestHashText(t *testing.T) {
	h := hashText("hello world")
	if len(h) != SignatureTextHashLen {
		t.Errorf("expected hash length %d, got %d", SignatureTextHashLen, len(h))
	}
	if h2 := hashText("hello world"); h != h2 {
		t.Errorf("hash not deterministic")
	}
}
