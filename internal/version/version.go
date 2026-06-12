package version

// Version is the default for local/dev builds ("dev").
// Official release versions are injected only via GitHub Release tag:
// -ldflags "-X github.com/wzhejunqiu/ds-code/internal/version.Version=<tag>"
var Version = "dev"
