package apistation

import (
	"strings"
	"testing"
)

// containsZeroWidth 检查字符串是否包含零宽字符
func containsZeroWidth(s string) bool {
	for _, r := range s {
		if r == '\u200B' || r == '\u200C' || r == '\u200D' || r == '\uFEFF' {
			return true
		}
	}
	return false
}

// stripZeroWidth 移除所有零宽字符
func stripZeroWidth(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r != '\u200B' && r != '\u200C' && r != '\u200D' && r != '\uFEFF' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestObfuscateWord(t *testing.T) {
	result := ObfuscateWord("proxy")
	if !containsZeroWidth(result) {
		t.Error("expected zero-width chars in result")
	}
	if stripZeroWidth(result) != "proxy" {
		t.Errorf("stripped result should be 'proxy', got %q", stripZeroWidth(result))
	}
}

func TestObfuscateWord_Empty(t *testing.T) {
	if ObfuscateWord("") != "" {
		t.Error("empty input should return empty")
	}
}

func TestObfuscateWord_SingleChar(t *testing.T) {
	result := ObfuscateWord("a")
	if result != "a" {
		t.Errorf("single char should not be modified, got %q", result)
	}
}

func TestObfuscateText(t *testing.T) {
	text := "This uses sub2api as a proxy"
	words := []string{"sub2api", "proxy"}
	result := ObfuscateText(text, words)

	// 原词应该被混淆
	if strings.Contains(result, "sub2api") {
		t.Error("sub2api should be obfuscated")
	}
	if strings.Contains(strings.ToLower(result), "proxy") && !containsZeroWidth(result) {
		t.Error("proxy should be obfuscated")
	}

	// 去掉零宽字符后应该恢复原文
	if stripZeroWidth(result) != text {
		t.Errorf("stripped should equal original, got %q", stripZeroWidth(result))
	}
}

func TestObfuscateText_CaseInsensitive(t *testing.T) {
	text := "PROXY and Proxy"
	result := ObfuscateText(text, []string{"proxy"})
	stripped := stripZeroWidth(result)
	if stripped != text {
		t.Errorf("case should be preserved, got %q", stripped)
	}
}

func TestObfuscateText_Chinese(t *testing.T) {
	text := "这是一个代理服务"
	result := ObfuscateText(text, []string{"代理"})
	if stripZeroWidth(result) != text {
		t.Errorf("chinese obfuscation failed, got %q", stripZeroWidth(result))
	}
	if !containsZeroWidth(result) {
		t.Error("should contain zero-width chars")
	}
}

func TestObfuscateBody(t *testing.T) {
	body := []byte(`{"system":"hello proxy world","messages":[{"role":"user","content":"use sub2api"}]}`)
	words := []string{"proxy", "sub2api"}
	result := ObfuscateBody(body, words)
	resultStr := string(result)

	// 零宽字符应该存在
	if !containsZeroWidth(resultStr) {
		t.Error("body should contain zero-width chars")
	}

	// 去掉零宽字符后原文应该保留
	stripped := stripZeroWidth(resultStr)
	if !strings.Contains(stripped, "proxy") || !strings.Contains(stripped, "sub2api") {
		t.Error("stripped body should contain original words")
	}
}

func TestObfuscateBody_Empty(t *testing.T) {
	if result := ObfuscateBody(nil, []string{"test"}); result != nil {
		t.Error("nil body should return nil")
	}
	if result := ObfuscateBody([]byte(`{}`), nil); string(result) != "{}" {
		t.Error("nil words should return original")
	}
}
