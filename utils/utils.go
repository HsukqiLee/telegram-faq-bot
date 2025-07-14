package utils

// Min 返回两个整数中的较小值
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetMatchTypeText 根据匹配类型返回对应的文本描述
func GetMatchTypeText(matchType int) string {
	switch matchType {
	case 1:
		return "精确"
	case 2:
		return "模糊"
	case 3:
		return "正则"
	default:
		return "未知"
	}
}
