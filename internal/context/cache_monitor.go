package context

import (
	"sync"

	"github.com/wzhejunqiu/ds-code/internal/logging"
	"go.uber.org/zap"
)

const cacheHitDropRatio = 0.15

type promptUsageEntry struct {
	staticHash   string
	promptTokens int
}

// promptUsage tracks last request usage per session for cache-hit heuristics.
type promptUsage struct {
	mu   sync.Mutex
	byID map[string]promptUsageEntry
}

func newPromptUsage() *promptUsage {
	return &promptUsage{byID: make(map[string]promptUsageEntry)}
}

// RecordPromptUsage logs a possible prompt cache hit when static prefix is unchanged
// but reported prompt_tokens dropped significantly.
func (s *Service) RecordPromptUsage(sessionID string, promptTokens int) {
	if s == nil || sessionID == "" || promptTokens <= 0 {
		return
	}
	if s.promptUsage == nil {
		s.promptUsage = newPromptUsage()
	}
	s.promptUsage.mu.Lock()
	prev := s.promptUsage.byID[sessionID]
	staticHash := prev.staticHash
	s.promptUsage.byID[sessionID] = promptUsageEntry{
		staticHash:   staticHash,
		promptTokens: promptTokens,
	}
	s.promptUsage.mu.Unlock()

	if staticHash == "" {
		return
	}
	if prev.promptTokens <= 0 {
		return
	}
	drop := float64(prev.promptTokens-promptTokens) / float64(prev.promptTokens)
	if drop >= cacheHitDropRatio {
		logging.L().Info("possible_prompt_cache_hit",
			zap.String("session_id", sessionID),
			zap.String("static_hash", staticHash),
			zap.Int("prev_prompt_tokens", prev.promptTokens),
			zap.Int("prompt_tokens", promptTokens),
			zap.Float64("drop_ratio", drop),
		)
	}
}

// noteStaticHash records the static prefix hash for a session (called from PrepareRequest).
func (s *Service) noteStaticHash(sessionID, staticHash string) {
	if s == nil || sessionID == "" || staticHash == "" {
		return
	}
	if s.promptUsage == nil {
		s.promptUsage = newPromptUsage()
	}
	s.promptUsage.mu.Lock()
	entry := s.promptUsage.byID[sessionID]
	entry.staticHash = staticHash
	s.promptUsage.byID[sessionID] = entry
	s.promptUsage.mu.Unlock()
}
