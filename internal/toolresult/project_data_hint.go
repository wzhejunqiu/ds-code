package toolresult

import "fmt"

// SavedResultHint formats the spill path hint appended to truncated tool/agent results.
func SavedResultHint(displayPath string) string {
	return fmt.Sprintf("\n... [完整结果已保存至 %s；请用 read_file 读取该绝对路径（shell 无法访问）]", displayPath)
}
