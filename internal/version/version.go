package version

// Version is the ds-code release version (single source of truth).
// Release builds may override via -ldflags "-X .../internal/version.Version=...".
var Version = "0.1.0-dev"
