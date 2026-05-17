package role

// Role is an OpenAI-compatible chat message role (session DB + LLM API wire format).
type Role string

const (
	User      Role = "user"      // end-user prompt
	Assistant Role = "assistant" // model reply (may include tool_calls)
	Tool      Role = "tool"      // tool result row
	System    Role = "system"    // history-only events (checkpoint rewind, turn cancel, merged system is separate)
)

// String returns the wire/storage representation.
func (r Role) String() string {
	return string(r)
}
