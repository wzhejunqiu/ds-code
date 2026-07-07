package sys

// Hooks are optional callbacks wired from cmd/ds-code-desktop on startup.
var Hooks struct {
	UpdateBadge func(count int)
	Notify      func(title, body string)
}

func refreshBadge() {
	if Hooks.UpdateBadge != nil {
		Hooks.UpdateBadge(BadgeCount())
	}
}

// TurnStarted marks a turn as running and updates badge.
func TurnStarted() {
	IncRunningTurn()
	refreshBadge()
}

// TurnFinished marks a turn as done and optionally notifies.
func TurnFinished(notifyTitle, notifyBody string, shouldNotify bool) {
	DecRunningTurn()
	refreshBadge()
	if shouldNotify && notifyBody != "" && Hooks.Notify != nil {
		Hooks.Notify(notifyTitle, notifyBody)
	}
}

// PermissionWaiting updates permission wait state.
func PermissionWaiting(waiting bool) {
	SetWaitingPermission(waiting)
	refreshBadge()
	if waiting && Hooks.Notify != nil {
		Hooks.Notify("ds-code", "Waiting for permission approval")
	}
}
