package normalize

import "strings"

// normalizeText 标准化用户文本：转小写、去首尾空格、合并连续空格
// 用于缓存 key 生成，覆盖 ASR 微小识别差异
func normalizeText(text string) string {
	t := strings.ToLower(strings.TrimSpace(text))
	return strings.Join(strings.Fields(t), " ")
}
