package apistation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/tidwall/gjson"
)

// DefaultFingerprintSalt — Claude Code 默认指纹盐值
// 对应 auth2api cloaking.ts line 18
const DefaultFingerprintSalt = "59cf53e54c78"

// DefaultCLIVersion — 当无 SettingKey 配置时的 fallback 版本
const DefaultCLIVersion = "1.0.33"

// ComputeFingerprint 计算消息指纹，精确复刻 Claude Code 的 utils/fingerprint.ts
// 算法: SHA256(SALT + msg[4] + msg[7] + msg[20] + version).slice(0, 3)
func ComputeFingerprint(messageText, version, salt string) string {
	if salt == "" {
		salt = DefaultFingerprintSalt
	}
	indices := [3]int{4, 7, 20}
	chars := make([]byte, 0, 3)
	for _, i := range indices {
		if i < len(messageText) {
			chars = append(chars, messageText[i])
		} else {
			chars = append(chars, '0')
		}
	}
	input := fmt.Sprintf("%s%s%s", salt, string(chars), version)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])[:3]
}

// ExtractFirstUserMessageText 从请求 body 的 messages 数组中提取第一条 user 消息文本
// 使用 gjson 避免完整反序列化
func ExtractFirstUserMessageText(body []byte) string {
	messages := gjson.GetBytes(body, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return ""
	}
	var text string
	messages.ForEach(func(_, msg gjson.Result) bool {
		if msg.Get("role").String() != "user" {
			return true // continue
		}
		// content 可能是 string 或 array
		content := msg.Get("content")
		if content.Type == gjson.String {
			text = content.String()
			return false // found
		}
		if content.IsArray() {
			content.ForEach(func(_, block gjson.Result) bool {
				if block.Get("type").String() == "text" {
					text = block.Get("text").String()
					return false
				}
				return true
			})
			if text != "" {
				return false
			}
		}
		return true
	})
	return text
}
