package slash

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// WriteHelp prints all registered commands to w.
func WriteHelp(w io.Writer) {
	cmds := All()
	sort.Slice(cmds, func(i, j int) bool { return cmds[i].Name < cmds[j].Name })

	var b strings.Builder
	b.WriteString("Slash commands (line must start with /):\n\n")
	for _, c := range cmds {
		line := fmt.Sprintf("  /%-14s", c.Name)
		if c.Args != "" {
			line += c.Args + " "
		} else {
			line += " "
		}
		line += c.Description
		if c.Phase != "" {
			line += fmt.Sprintf(" [Phase %s]", c.Phase)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	_, _ = io.WriteString(w, b.String())
}
