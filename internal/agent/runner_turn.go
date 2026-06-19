package agent

import (
	"context"
	"fmt"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"go.uber.org/zap"
	"time"
)

type runTurnOptions struct {
	appendUser bool
}

// RunTurn handles one user message through sub-rounds until no tool_calls or max turns.
func (r *Runner) RunTurn(ctx context.Context, sessionID, userText string, cb *TurnCallbacks) (*TurnResult, error) {
	return r.runTurn(ctx, sessionID, userText, cb, runTurnOptions{appendUser: true})
}

// RunTurnSeeded runs a turn without appending a user message (fork child with pre-seeded history).
func (r *Runner) RunTurnSeeded(ctx context.Context, sessionID string, cb *TurnCallbacks) (*TurnResult, error) {
	return r.runTurn(ctx, sessionID, "", cb, runTurnOptions{appendUser: false})
}

func (r *Runner) runTurn(ctx context.Context, sessionID, userText string, cb *TurnCallbacks, opts runTurnOptions) (*TurnResult, error) {
	ctx = WithActiveTurn(ctx)
	defer WithoutActiveTurn(ctx)
	if r.Perm != nil {
		r.Perm.SpillSessionID = sessionID
		if r.Cfg != nil && r.Cfg.ProjectRoot != "" {
			r.Perm.ProjectRoot = r.Cfg.ProjectRoot
		}
	}
	if cb != nil {
		ctx = WithTurnCallbacks(ctx, cb)
	}
	if opts.appendUser {
		logging.L().Info("user turn start", zap.String("session_id", sessionID), zap.Int("chars", len(userText)))
	} else {
		logging.L().Info("seeded turn start", zap.String("session_id", sessionID))
	}

	if opts.appendUser && r.DrainNotifications != nil {
		if notice := r.DrainNotifications(ctx); notice != "" {
			userText = notice + "\n" + userText
		}
	}

	if opts.appendUser {
		existing, listErr := r.Sessions.ListMessages(ctx, sessionID)
		if listErr != nil {
			return nil, listErr
		}
		isFirstUser := len(existing) == 0
		expanded, err := r.Context.ExpandUserText(userText)
		if err != nil {
			return nil, fmt.Errorf("expand @ references: %w", err)
		}
		if err := r.Sessions.AppendMessage(ctx, session.Message{
			SessionID: sessionID,
			Role:      role.User,
			Content:   expanded,
		}); err != nil {
			return nil, err
		}
		if r.Hooks != nil && isFirstUser {
			if r.sessionStarted == nil {
				r.sessionStarted = make(map[string]bool)
			}
			if !r.sessionStarted[sessionID] {
				r.sessionStarted[sessionID] = true
				r.Hooks.Run(ctx, HookSessionStart, marshalHookInput(HookInput{SessionID: sessionID}))
			}
		}
	}

	sess, err := r.Sessions.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	r.Context.BeginUserTurn()
	defer r.Context.EndUserTurn()

	turnStart := time.Now()
	result := &TurnResult{}
	state := &LoopState{}
	for round := 0; round < r.MaxTurns; round++ {
		state.Round = round
		state.Phase = PhasePrepare
		if ctx.Err() != nil {
			if r.Hooks != nil {
				r.Hooks.Run(ctx, HookStop, marshalHookInput(HookInput{SessionID: sessionID, Error: ctx.Err().Error()}))
			}
			return nil, ctx.Err()
		}
		if round > 0 && cb != nil && cb.OnAssistantSegmentEnd != nil {
			cb.OnAssistantSegmentEnd()
		}
		planningFromAgent := false
		if round > 0 && cb != nil && cb.OnPlanningStart != nil {
			cb.OnPlanningStart()
			planningFromAgent = true
		}

		logging.L().Debug("agent sub-round", zap.String("session_id", sessionID), zap.Int("round", round+1))
		view, maxTokens, err := r.Context.PrepareRequest(ctx, sessionID)
		if err != nil {
			if planningFromAgent && cb.OnPlanningEnd != nil {
				cb.OnPlanningEnd()
			}
			return nil, err
		}

		req := llm.Request{
			MergedSystem:    view.MergedSystem(),
			Messages:        view.Messages,
			Model:           sess.Model,
			Tools:           r.Tools.Definitions(),
			MaxTokens:       maxTokens,
			Stream:          true,
			ThinkingType:    sess.ThinkingType,
			ReasoningEffort: sess.ReasoningEffort,
			UserID:          cacheScope(sessionID),
			StrictTools:     r.Cfg.LLM.StrictTools,
		}
		stream := &subRoundStream{}
		req.OnStream = r.attachStreamHandlers(cb, round, stream)

		resp, err := r.chatWithRecovery(ctx, sessionID, req, state)
		if err != nil {
			if !stream.planningDone && cb != nil && cb.OnPlanningEnd != nil {
				cb.OnPlanningEnd()
			}
			logging.L().Error("LLM request failed", zap.String("session_id", sessionID), zap.Error(err))
			return nil, err
		}
		if !stream.planningDone && cb != nil && cb.OnPlanningEnd != nil {
			cb.OnPlanningEnd()
		}
		logging.L().Debug("LLM response",
			zap.String("session_id", sessionID),
			zap.Int("tool_calls", len(resp.ToolCalls)),
			zap.Int("content_chars", len(resp.Content)),
		)

		r.applySubRoundUsage(ctx, sessionID, resp.Usage, cb)
		result.Usage = resp.Usage
		result.SubRounds = round + 1

		if len(resp.ToolCalls) == 0 {
			return r.finishTerminalRound(ctx, sessionID, sess.Model, resp, stream, turnStart, result, cb, HookInput{SessionID: sessionID})
		}
		if err := r.appendAssistantWithTools(ctx, sessionID, sess.Model, resp, stream); err != nil {
			return nil, err
		}
		if err := r.runToolCalls(ctx, sessionID, resp.ToolCalls, resp, stream, cb); err != nil {
			return nil, err
		}
		if r.DrainNotificationsLater != nil {
			r.DrainNotificationsLater(ctx, sessionID)
		}
	}
	return r.finishMaxTurnsExceeded(ctx, sessionID, sess, turnStart, result, state, cb)
}

// DrainNotificationsLaterFunc appends PrioLater async agent notices to the main session.
type DrainNotificationsLaterFunc func(ctx context.Context, sessionID string)
