package permission

// PromptRequest is sent to the TUI when ask mode needs approval.
type PromptRequest struct {
	Tool    string
	Summary string
	Reply   chan bool
}

// TUIPrompter blocks until the TUI responds on Reply.
func TUIPrompter(reqCh chan<- PromptRequest) Prompter {
	return func(tool, summary string) (bool, error) {
		reply := make(chan bool, 1)
		reqCh <- PromptRequest{Tool: tool, Summary: summary, Reply: reply}
		return <-reply, nil
	}
}
