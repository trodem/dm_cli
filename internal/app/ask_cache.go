package app

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"cli/internal/agent"
)

type decisionCacheEntry struct {
	at    time.Time
	value agent.DecisionResult
}

type decisionCacheStore struct {
	mu    sync.RWMutex
	ttl   time.Duration
	items map[string]decisionCacheEntry
}

func newDecisionCacheStore(ttl time.Duration) *decisionCacheStore {
	return &decisionCacheStore{
		ttl:   ttl,
		items: map[string]decisionCacheEntry{},
	}
}

func (c *decisionCacheStore) Get(key string, now time.Time) (agent.DecisionResult, bool) {
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return agent.DecisionResult{}, false
	}
	if now.Sub(entry.at) > c.ttl {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return agent.DecisionResult{}, false
	}
	return entry.value, true
}

func (c *decisionCacheStore) Set(key string, value agent.DecisionResult, now time.Time) {
	c.mu.Lock()
	c.items[key] = decisionCacheEntry{
		at:    now,
		value: value,
	}
	c.mu.Unlock()
}

var askDecisionCache = newDecisionCacheStore(askDecisionCacheTTL)

func decideWithCache(prompt, pluginCatalog, toolCatalog string, opts agent.AskOptions, envContext string) (agent.DecisionResult, bool, error) {
	key := decisionCacheKey(prompt, pluginCatalog, toolCatalog, opts, envContext)
	now := time.Now()
	if cached, ok := askDecisionCache.Get(key, now); ok {
		return cached, true, nil
	}
	decision, err := agent.DecideWithPlugins(prompt, pluginCatalog, toolCatalog, opts, envContext)
	if err != nil {
		return agent.DecisionResult{}, false, err
	}
	askDecisionCache.Set(key, decision, now)
	return decision, false, nil
}

func decisionCacheKey(prompt, pluginCatalog, toolCatalog string, opts agent.AskOptions, envContext string) string {
	normalized := strings.Join([]string{
		strings.TrimSpace(prompt),
		strings.TrimSpace(pluginCatalog),
		strings.TrimSpace(toolCatalog),
		strings.ToLower(strings.TrimSpace(opts.Provider)),
		strings.TrimSpace(opts.Model),
		strings.TrimSpace(opts.BaseURL),
		strings.TrimSpace(envContext),
	}, "\n---\n")
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
