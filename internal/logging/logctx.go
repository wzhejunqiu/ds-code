package logging

import (
	"context"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var (
	logctxMu     sync.Mutex
	logctxStacks = map[uint64][]context.Context{}
	logctxActive bool
)

// SetLogctxActive enables logctx stack management (tracing on).
func SetLogctxActive(active bool) {
	logctxMu.Lock()
	logctxActive = active
	if !active {
		logctxStacks = map[uint64][]context.Context{}
	}
	logctxMu.Unlock()
}

func currentGoID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	// "goroutine 123 [running]:\n..."
	idStr := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, _ := strconv.ParseUint(idStr, 10, 64)
	return id
}

// Push records ctx on the current goroutine stack; pop is returned for defer.
func Push(ctx context.Context) (pop func()) {
	if !logctxActive {
		return func() {}
	}
	id := currentGoID()
	logctxMu.Lock()
	logctxStacks[id] = append(logctxStacks[id], ctx)
	logctxMu.Unlock()
	return func() {
		logctxMu.Lock()
		stack := logctxStacks[id]
		if len(stack) == 0 {
			logctxMu.Unlock()
			return
		}
		if len(stack) == 1 {
			delete(logctxStacks, id)
		} else {
			logctxStacks[id] = stack[:len(stack)-1]
		}
		logctxMu.Unlock()
	}
}

// Current returns the top context on this goroutine's stack, or nil.
func Current() context.Context {
	if !logctxActive {
		return nil
	}
	id := currentGoID()
	logctxMu.Lock()
	defer logctxMu.Unlock()
	stack := logctxStacks[id]
	if len(stack) == 0 {
		return nil
	}
	return stack[len(stack)-1]
}

// Bind copies the ctx stack from the caller goroutine into a new goroutine.
// Use at goroutine entry: defer logctx.Bind(ctx)().
func Bind(ctx context.Context) (pop func()) {
	return Push(ctx)
}
