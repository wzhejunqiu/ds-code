package patch

import (
	"fmt"

	wspkg "github.com/hejunqiu/ds-code/internal/workspace"
)

// ValidatePath resolves rel under workspace and ensures the result stays inside workspace.
func ValidatePath(workspace, rel string) error {
	if workspace == "" {
		return nil
	}
	if err := wspkg.ValidateRel(workspace, rel); err != nil {
		if rel != "" {
			return fmt.Errorf("patch: path outside workspace: %s", rel)
		}
		return fmt.Errorf("patch: %w", err)
	}
	return nil
}
