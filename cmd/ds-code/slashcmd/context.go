package slashcmd

import (
	"fmt"
	"strings"

	uipkg "github.com/wzhejunqiu/ds-code/internal/ui"
)

func Context(env *Env, args string) error {
	view, err := env.CtxSvc.BuildAPIContext(env.Ctx, *env.SessionID)
	if err != nil {
		return err
	}
	sess, err := env.Store.Get(env.Ctx, *env.SessionID)
	if err != nil {
		return err
	}
	panel, err := uipkg.BuildContextPanelData(env.Ctx, env.Cfg, env.Store, env.CtxSvc.Subagent, sess, view)
	if err != nil {
		return err
	}
	if strings.TrimSpace(args) == "--json" {
		text, err := uipkg.FormatContextJSON(panel)
		if err != nil {
			return err
		}
		fmt.Fprintln(env.Out, text)
		return nil
	}
	fmt.Fprintln(env.Out, uipkg.FormatContextPanel(panel))
	return nil
}
