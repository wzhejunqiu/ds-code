package bridge

import (
	"sync"
	"time"
)

const (
	defaultFlushInterval = 16 * time.Millisecond
	defaultMaxChunk      = 8192
	eventRetryInterval   = 5 * time.Millisecond
	eventMaxRetries      = 200
)

// EmitFunc delivers an envelope to the UI layer (e.g. Wails Events).
type EmitFunc func(env AgentEventEnvelope) bool

type streamBuffer struct {
	mu               sync.Mutex
	pendingContent   string
	pendingReasoning string
}

func (b *streamBuffer) appendContent(delta string) {
	b.mu.Lock()
	b.pendingContent += delta
	b.mu.Unlock()
}

func (b *streamBuffer) appendReasoning(delta string) {
	b.mu.Lock()
	b.pendingReasoning += delta
	b.mu.Unlock()
}

func (b *streamBuffer) takeContent(max int) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.pendingContent == "" {
		return ""
	}
	if max > 0 && len(b.pendingContent) > max {
		out := b.pendingContent[:max]
		b.pendingContent = b.pendingContent[max:]
		return out
	}
	out := b.pendingContent
	b.pendingContent = ""
	return out
}

func (b *streamBuffer) takeReasoning(max int) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.pendingReasoning == "" {
		return ""
	}
	if max > 0 && len(b.pendingReasoning) > max {
		out := b.pendingReasoning[:max]
		b.pendingReasoning = b.pendingReasoning[max:]
		return out
	}
	out := b.pendingReasoning
	b.pendingReasoning = ""
	return out
}

func (b *streamBuffer) hasPending() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.pendingContent != "" || b.pendingReasoning != ""
}

// StreamEmitter batches content/reasoning deltas before emitting to Wails.
type StreamEmitter struct {
	turnID      string
	streamID    string
	workspaceID string
	seq         uint64
	buf         streamBuffer
	emit        EmitFunc
	flushEvery  time.Duration
	maxChunk    int
	lastFlush   time.Time
	mu          sync.Mutex
}

// StreamEmitterOptions configures batching behavior.
type StreamEmitterOptions struct {
	TurnID      string
	StreamID    string
	WorkspaceID string
	Emit        EmitFunc
	FlushEvery  time.Duration
	MaxChunk    int
}

// NewStreamEmitter creates an emitter with PoC defaults (16ms / 8KB).
func NewStreamEmitter(opts StreamEmitterOptions) *StreamEmitter {
	flush := opts.FlushEvery
	if flush <= 0 {
		flush = defaultFlushInterval
	}
	maxChunk := opts.MaxChunk
	if maxChunk <= 0 {
		maxChunk = defaultMaxChunk
	}
	streamID := opts.StreamID
	if streamID == "" {
		streamID = "main"
	}
	workspaceID := opts.WorkspaceID
	if workspaceID == "" {
		workspaceID = "default"
	}
	return &StreamEmitter{
		turnID:      opts.TurnID,
		streamID:    streamID,
		workspaceID: workspaceID,
		emit:        opts.Emit,
		flushEvery:  flush,
		maxChunk:    maxChunk,
		lastFlush:   time.Now(),
	}
}

func (s *StreamEmitter) nextSeq() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return s.seq
}

func (s *StreamEmitter) envelope(kind AgentEventKind, critical bool, payload any) AgentEventEnvelope {
	return AgentEventEnvelope{
		V:           EnvelopeVersion,
		Seq:         s.nextSeq(),
		TurnID:      s.turnID,
		StreamID:    s.streamID,
		WorkspaceID: s.workspaceID,
		Kind:        kind,
		Ts:          time.Now().UnixMilli(),
		Critical:    critical,
		Payload:     mustPayload(payload),
	}
}

func (s *StreamEmitter) send(env AgentEventEnvelope, critical bool) {
	if s.emit == nil {
		return
	}
	if !critical {
		s.emit(env)
		return
	}
	for i := 0; i < eventMaxRetries; i++ {
		if s.emit(env) {
			return
		}
		time.Sleep(eventRetryInterval)
	}
}

// EmitCritical sends a critical envelope with retry semantics.
func (s *StreamEmitter) EmitCritical(kind AgentEventKind, payload any) {
	s.send(s.envelope(kind, true, payload), true)
}

// EmitNonCritical sends a non-critical envelope; drops when the sink is full.
func (s *StreamEmitter) EmitNonCritical(kind AgentEventKind, payload any) {
	s.send(s.envelope(kind, false, payload), false)
}

// OnDelta appends streaming text and flushes when interval or chunk threshold is met.
func (s *StreamEmitter) OnDelta(kind AgentEventKind, delta string) {
	if delta == "" {
		return
	}
	switch kind {
	case KindContentDelta:
		s.buf.appendContent(delta)
	case KindReasoningDelta:
		s.buf.appendReasoning(delta)
	default:
		return
	}
	s.maybeFlush(false)
}

func (s *StreamEmitter) maybeFlush(force bool) {
	s.mu.Lock()
	elapsed := time.Since(s.lastFlush)
	s.mu.Unlock()
	if !force && elapsed < s.flushEvery && !s.chunkReady() {
		return
	}
	s.Flush(false)
}

func (s *StreamEmitter) chunkReady() bool {
	s.buf.mu.Lock()
	defer s.buf.mu.Unlock()
	return len(s.buf.pendingContent) >= s.maxChunk || len(s.buf.pendingReasoning) >= s.maxChunk
}

// Flush emits buffered content/reasoning deltas.
func (s *StreamEmitter) Flush(force bool) {
	for {
		content := s.buf.takeContent(s.maxChunk)
		if content != "" {
			s.EmitNonCritical(KindContentDelta, ContentDeltaPayload{Delta: content})
		}
		reasoning := s.buf.takeReasoning(s.maxChunk)
		if reasoning != "" {
			s.EmitNonCritical(KindReasoningDelta, ReasoningDeltaPayload{Delta: reasoning})
		}
		if content == "" && reasoning == "" {
			break
		}
		if !force && !s.buf.hasPending() {
			break
		}
	}
	s.mu.Lock()
	s.lastFlush = time.Now()
	s.mu.Unlock()
}

// Seq returns the current sequence number (for tests).
func (s *StreamEmitter) Seq() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seq
}
