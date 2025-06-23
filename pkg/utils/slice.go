package utils

import (
	"strconv"
	"strings"
)

func SliceContains[T comparable](slice []T, value T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// SliceToString 切片转字符串
func SliceToString(slice []int32) string {
	var strs []string
	for _, v := range slice {
		strs = append(strs, strconv.FormatInt(int64(v), 10))
	}
	return strings.Join(strs, ",")
}

func DeDuplicate(slice []string) []string {
	seen := make(map[string]struct{}) // 使用空结构体减少内存占用
	result := make([]string, 0, len(slice))

	for _, s := range slice {
		if _, exists := seen[s]; !exists {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}
	return result
}
