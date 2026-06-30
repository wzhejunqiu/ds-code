package permission

import (
	"errors"
	"strings"
)

func classifyDeny(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, ErrRejected):
		return "user_rejected"
	case errors.Is(err, ErrNeedTTY):
		return "ask_no_tty"
	case errors.Is(err, ErrDenied):
		msg := strings.ToLower(err.Error())
		switch {
		case strings.Contains(msg, "readonly"):
			return "readonly"
		case strings.Contains(msg, "sensitive"):
			return "sensitive_path"
		case strings.Contains(msg, "no prompter"):
			return "ask_no_prompter"
		case strings.Contains(msg, "invalid patch"):
			return "invalid_patch"
		case strings.Contains(msg, "allowlist"):
			return "allowlist"
		default:
			return "denied"
		}
	default:
		return "error"
	}
}
