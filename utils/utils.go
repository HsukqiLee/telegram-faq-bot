package utils

import "TGFaqBot/database"

// Min 返回两个整数中的较小值
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetMatchTypeText 根据匹配类型返回对应的文本描述
func GetMatchTypeText(matchType database.MatchType) string {
	return matchType.String()
}

// GetMatchTypeFromInt 从整数获取匹配类型（为了向后兼容）
func GetMatchTypeFromInt(i int) database.MatchType {
	mt, _ := database.MatchTypeFromInt(i)
	return mt
}
