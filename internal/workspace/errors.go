package workspace

import "errors"

// ErrOutsideWorkspace is returned when a resolved path leaves the workspace jail.
var ErrOutsideWorkspace = errors.New("workspace: path outside workspace")
