package ssh

import "strings"

// ShellQuote 对字符串做 Shell 安全转义（单引号包裹）
// 防止命令注入: 用户输入中的特殊字符被安全处理
func ShellQuote(s string) string {
	// 单引号内所有字符都是字面量，只需处理单引号本身
	// 方式: 'foo'\''bar' => foo'bar
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
