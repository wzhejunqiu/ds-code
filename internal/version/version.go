package version

// Name is the CLI binary and protocol client application name.
const Name = "ds-code"

// UserDataDirName is the basename of user/project metadata directories (".ds-code").
const UserDataDirName = "." + Name

// SystemTag is the bracketed name for persisted system event messages ("[ds-code]").
const SystemTag = "[" + Name + "]"

// SystemPrefix is SystemTag followed by a space.
const SystemPrefix = SystemTag + " "

// Version is the default for local/dev builds ("dev").
// Official release versions are injected only via GitHub Release tag:
// -ldflags "-X github.com/wzhejunqiu/ds-code/internal/version.Version=<tag>"
var Version = "dev"
