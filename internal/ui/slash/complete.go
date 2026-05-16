package slash

import "strings"

// FilterCommands returns commands matching an input prefix like "/comp" or "/".
func FilterCommands(input string) []Command {
	input = strings.TrimSpace(input)
	if input == "" || input[0] != '/' {
		return nil
	}
	input = strings.TrimPrefix(input, "/")
	space := strings.Index(input, " ")
	if space >= 0 {
		input = input[:space]
	}
	prefix := strings.ToLower(input)

	var out []Command
	for _, c := range All() {
		if c.Phase != "" {
			// still offer for discovery; execution may defer to stub
		}
		if prefix == "" || strings.HasPrefix(c.Name, prefix) {
			out = append(out, c)
		}
	}
	return out
}
