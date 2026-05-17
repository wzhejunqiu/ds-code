package patch

// ChangeKind is a file operation in a Codex-style patch document.
type ChangeKind int

const (
	ChangeAdd ChangeKind = iota // *** Add File:
	ChangeDelete                // *** Delete File:
	ChangeUpdate                // *** Update File:
)

func (k ChangeKind) String() string {
	switch k {
	case ChangeAdd:
		return "add"
	case ChangeDelete:
		return "delete"
	case ChangeUpdate:
		return "update"
	default:
		return "unknown"
	}
}
