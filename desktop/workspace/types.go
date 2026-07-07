package workspace

// Summary is a workspace row for UI listing.
type Summary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Root         string `json:"root"`
	Active       bool   `json:"active"`
	LastOpenedAt int64  `json:"lastOpenedAt"`
	Valid        bool   `json:"valid"`
}

// ChatSummary is an Agent conversation window row.
type ChatSummary struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Model     string `json:"model"`
	UpdatedAt int64  `json:"updatedAt"`
	CreatedAt int64  `json:"createdAt"`
}

// ChatMessage is a history row for resume rendering.
type ChatMessage struct {
	ID         int64  `json:"id"`
	Role       string `json:"role"`
	Content    string `json:"content"`
	Reasoning  string `json:"reasoning,omitempty"`
	ToolCalls  string `json:"toolCalls,omitempty"`
	ToolCallID string `json:"toolCallId,omitempty"`
	ToolName   string `json:"toolName,omitempty"`
	CreatedAt  int64  `json:"createdAt"`
}
