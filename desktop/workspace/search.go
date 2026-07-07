package workspace

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	desktopdatadir "github.com/wzhejunqiu/ds-code/desktop/datadir"
	_ "modernc.org/sqlite"
)

// SearchChats finds sessions by title or message content.
func (m *Manager) SearchChats(wsID, query string) ([]ChatSummary, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return m.ListChats(wsID)
	}
	root, err := m.ProjectRoot(wsID)
	if err != nil {
		return nil, err
	}
	dbPath := desktopdatadir.DefaultDBPath(root)
	if dbPath == "" {
		return nil, fmt.Errorf("database path unavailable")
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	pattern := "%" + query + "%"
	rows, err := db.Query(`
SELECT DISTINCT s.id, s.title, s.model, s.run_mode, s.updated_at, s.created_at
FROM sessions s
LEFT JOIN messages m ON m.session_id = s.id
WHERE s.title LIKE ? OR m.content LIKE ?
ORDER BY s.updated_at DESC
LIMIT 50`, pattern, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ChatSummary
	for rows.Next() {
		var (
			id, title, model, runMode, updatedAt, createdAt string
		)
		if err := rows.Scan(&id, &title, &model, &runMode, &updatedAt, &createdAt); err != nil {
			return nil, err
		}
		if title == "" {
			title = "(untitled)"
		}
		if strings.HasPrefix(title, "[deleted]") {
			continue
		}
		updated, _ := time.Parse(time.RFC3339, updatedAt)
		created, _ := time.Parse(time.RFC3339, createdAt)
		out = append(out, ChatSummary{
			ID:        id,
			Title:     title,
			Model:     model,
			RunMode:   runMode,
			UpdatedAt: updated.UnixMilli(),
			CreatedAt: created.UnixMilli(),
		})
	}
	return out, rows.Err()
}
