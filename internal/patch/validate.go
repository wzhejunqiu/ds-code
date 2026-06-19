package patch

import "fmt"

// PathValidator checks that a relative patch path is allowed.
type PathValidator func(rel string) error

// ValidatePath invokes validate on rel when non-nil.
func ValidatePath(validate PathValidator, rel string) error {
	if validate == nil {
		return nil
	}
	if err := validate(rel); err != nil {
		if rel != "" {
			return fmt.Errorf("patch: path outside workspace: %s", rel)
		}
		return fmt.Errorf("patch: %w", err)
	}
	return nil
}
