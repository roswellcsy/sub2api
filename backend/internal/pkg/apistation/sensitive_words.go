package apistation

import (
	"math/rand"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// 零宽字符集合（用于随机插入）
var zeroWidthChars = []string{
	"\u200B", // Zero-Width Space
	"\u200C", // Zero-Width Non-Joiner
	"\u200D", // Zero-Width Joiner
	"\uFEFF", // Zero-Width No-Break Space (BOM)
}

// DefaultSensitiveWords 默认敏感词列表
var DefaultSensitiveWords = []string{
	"sub2api",
	"proxy",
	"mirror",
	"reverse",
	"forwarding",
	"中转",
	"代理",
	"镜像",
}

// ObfuscateWord 在单词的每个字符之间随机插入零宽字符
func ObfuscateWord(word string) string {
	if len(word) == 0 {
		return word
	}
	var b strings.Builder
	runes := []rune(word)
	for i, r := range runes {
		b.WriteRune(r)
		if i < len(runes)-1 {
			// 在字符间插入 1 个随机零宽字符
			zwc := zeroWidthChars[rand.Intn(len(zeroWidthChars))]
			b.WriteString(zwc)
		}
	}
	return b.String()
}

// ObfuscateText 替换文本中的所有敏感词（大小写不敏感）
func ObfuscateText(text string, words []string) string {
	if len(text) == 0 || len(words) == 0 {
		return text
	}
	result := text
	lower := strings.ToLower(text)
	// 从后往前替换，避免偏移量问题
	type replacement struct {
		start, end int
		newText    string
	}
	var replacements []replacement
	for _, word := range words {
		if len(word) == 0 {
			continue
		}
		lowerWord := strings.ToLower(word)
		searchFrom := 0
		for {
			idx := strings.Index(lower[searchFrom:], lowerWord)
			if idx < 0 {
				break
			}
			absIdx := searchFrom + idx
			// 取原文中对应的子串（保持原始大小写）
			original := text[absIdx : absIdx+len(word)]
			replacements = append(replacements, replacement{
				start:   absIdx,
				end:     absIdx + len(word),
				newText: ObfuscateWord(original),
			})
			searchFrom = absIdx + len(word)
		}
	}
	if len(replacements) == 0 {
		return text
	}
	// 按位置从后往前替换
	for i := len(replacements) - 1; i >= 0; i-- {
		r := replacements[i]
		result = result[:r.start] + r.newText + result[r.end:]
	}
	return result
}

// ObfuscateBody 对请求 body 中 system 和 messages 内的敏感词进行零宽字符混淆
// 仅处理文本内容，不修改结构
func ObfuscateBody(body []byte, words []string) []byte {
	if len(body) == 0 || len(words) == 0 {
		return body
	}

	result := body

	// 1. 处理 system (可能是 string 或 array of text blocks)
	systemResult := gjson.GetBytes(result, "system")
	if systemResult.Exists() {
		switch systemResult.Type {
		case gjson.String:
			obfuscated := ObfuscateText(systemResult.String(), words)
			if obfuscated != systemResult.String() {
				if newBody, err := sjson.SetBytes(result, "system", obfuscated); err == nil {
					result = newBody
				}
			}
		case gjson.JSON:
			if systemResult.IsArray() {
				systemResult.ForEach(func(key, value gjson.Result) bool {
					if value.Get("type").String() == "text" {
						text := value.Get("text").String()
						obfuscated := ObfuscateText(text, words)
						if obfuscated != text {
							path := "system." + key.String() + ".text"
							if newBody, err := sjson.SetBytes(result, path, obfuscated); err == nil {
								result = newBody
							}
						}
					}
					return true
				})
			}
		}
	}

	// 2. 处理 messages[].content (string 或 array of text blocks)
	messages := gjson.GetBytes(result, "messages")
	if messages.Exists() && messages.IsArray() {
		messages.ForEach(func(msgKey, msg gjson.Result) bool {
			content := msg.Get("content")
			if !content.Exists() {
				return true
			}
			msgPath := "messages." + msgKey.String()
			switch content.Type {
			case gjson.String:
				obfuscated := ObfuscateText(content.String(), words)
				if obfuscated != content.String() {
					if newBody, err := sjson.SetBytes(result, msgPath+".content", obfuscated); err == nil {
						result = newBody
					}
				}
			case gjson.JSON:
				if content.IsArray() {
					content.ForEach(func(blockKey, block gjson.Result) bool {
						if block.Get("type").String() == "text" {
							text := block.Get("text").String()
							obfuscated := ObfuscateText(text, words)
							if obfuscated != text {
								path := msgPath + ".content." + blockKey.String() + ".text"
								if newBody, err := sjson.SetBytes(result, path, obfuscated); err == nil {
									result = newBody
								}
							}
						}
						return true
					})
				}
			}
			return true
		})
	}

	return result
}
