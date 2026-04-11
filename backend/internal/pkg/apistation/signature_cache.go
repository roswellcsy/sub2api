package apistation

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	SignatureCacheTTL    = 3 * time.Hour
	SignatureTextHashLen = 16
	MinValidSignatureLen = 50
	CacheCleanupInterval = 10 * time.Minute
)

// SignatureEntry holds a cached thinking signature with timestamp.
type SignatureEntry struct {
	Signature string
	CachedAt  time.Time
}

// groupCache is a per-model-group map of textHash -> SignatureEntry.
type groupCache struct {
	mu      sync.RWMutex
	entries map[string]SignatureEntry
}

// SignatureCache caches thinking block signatures by model group.
// Thread-safe. Used as a package-level singleton.
type SignatureCache struct {
	groups      sync.Map // string -> *groupCache
	cleanupOnce sync.Once
}

// DefaultSignatureCache is the package-level singleton.
var DefaultSignatureCache = &SignatureCache{}

func hashText(text string) string {
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:])[:SignatureTextHashLen]
}

func modelGroup(model string) string {
	// 简单分组: 不同 claude model 共享签名缓存
	return "claude"
}

func (c *SignatureCache) getOrCreateGroup(groupKey string) *groupCache {
	c.cleanupOnce.Do(func() {
		go c.cleanupLoop()
	})
	if val, ok := c.groups.Load(groupKey); ok {
		return val.(*groupCache)
	}
	gc := &groupCache{entries: make(map[string]SignatureEntry)}
	actual, _ := c.groups.LoadOrStore(groupKey, gc)
	return actual.(*groupCache)
}

func (c *SignatureCache) cleanupLoop() {
	ticker := time.NewTicker(CacheCleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.groups.Range(func(key, value any) bool {
			gc := value.(*groupCache)
			gc.mu.Lock()
			for k, e := range gc.entries {
				if now.Sub(e.CachedAt) > SignatureCacheTTL {
					delete(gc.entries, k)
				}
			}
			empty := len(gc.entries) == 0
			gc.mu.Unlock()
			if empty {
				c.groups.Delete(key)
			}
			return true
		})
	}
}

// Put caches a signature for a given model and thinking text.
func (c *SignatureCache) Put(model, text, signature string) {
	if text == "" || len(signature) < MinValidSignatureLen {
		return
	}
	gc := c.getOrCreateGroup(modelGroup(model))
	gc.mu.Lock()
	gc.entries[hashText(text)] = SignatureEntry{Signature: signature, CachedAt: time.Now()}
	gc.mu.Unlock()
}

// Get retrieves a cached signature. Returns "" if not found or expired.
func (c *SignatureCache) Get(model, text string) string {
	if text == "" {
		return ""
	}
	val, ok := c.groups.Load(modelGroup(model))
	if !ok {
		return ""
	}
	gc := val.(*groupCache)
	hash := hashText(text)

	gc.mu.RLock()
	entry, exists := gc.entries[hash]
	gc.mu.RUnlock()
	if !exists {
		return ""
	}
	if time.Since(entry.CachedAt) > SignatureCacheTTL {
		gc.mu.Lock()
		delete(gc.entries, hash)
		gc.mu.Unlock()
		return ""
	}
	return entry.Signature
}

// ProcessThinkingSignatures scans the request body for thinking blocks:
// 1. Blocks with valid signatures -> cache them
// 2. Blocks with missing/invalid signatures -> inject from cache
// Returns the (possibly modified) body.
func ProcessThinkingSignatures(model string, body []byte) []byte {
	messages := gjson.GetBytes(body, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return body
	}

	cache := DefaultSignatureCache
	messageArray := messages.Array()

	for _, msg := range messageArray {
		if msg.Get("role").String() != "assistant" {
			continue
		}
		content := msg.Get("content")
		if !content.Exists() || !content.IsArray() {
			continue
		}
		for _, block := range content.Array() {
			if block.Get("type").String() != "thinking" {
				continue
			}
			text := block.Get("thinking").String()
			sig := block.Get("signature").String()
			if text == "" || len(sig) < MinValidSignatureLen {
				continue
			}
			cache.Put(model, text, sig)
		}
	}

	result := body
	for mi, msg := range messageArray {
		if msg.Get("role").String() != "assistant" {
			continue
		}
		content := msg.Get("content")
		if !content.Exists() || !content.IsArray() {
			continue
		}
		for ci, block := range content.Array() {
			if block.Get("type").String() != "thinking" {
				continue
			}
			text := block.Get("thinking").String()
			sig := block.Get("signature").String()
			if text == "" || len(sig) >= MinValidSignatureLen {
				continue
			}
			cached := cache.Get(model, text)
			if cached == "" {
				continue
			}
			path := "messages." + strconv.Itoa(mi) + ".content." + strconv.Itoa(ci) + ".signature"
			newBody, err := sjson.SetBytes(result, path, cached)
			if err != nil {
				continue
			}
			result = newBody
		}
	}

	return result
}
