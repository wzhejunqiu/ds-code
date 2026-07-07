//go:build !darwin

package sys

// Notify is a no-op on non-macOS platforms.
func Notify(title, body string) {}

// SetDockBadge is a no-op on non-macOS platforms.
func SetDockBadge(count int) {}
