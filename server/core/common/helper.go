package common

import "net/url"

// Of 是一个泛型辅助函数，
// 接受任意类型的值 v，并返回其指针。
// 用于在需要指针的地方快速生成指针值。
func Of[T any](v T) *T {
	return &v
}

// IsURL 判断给定字符串是否为有效的 URL。
// 若解析成功且包含非空的 Scheme（协议）和 Host（主机），则返回 true。
func IsURL(str string) bool {
	u, err := url.Parse(str)
	if err != nil {
		return false
	}
	return u.Scheme != "" && u.Host != ""
}

// RemoveDuplicates 根据 keyFunc 指定的键函数，从切片中移除重复元素。
// 参数：
//   - slice: 原始切片。
//   - keyFunc: 用于从元素中提取可比较键的函数。
//
// 返回：
//   - 一个去重后的新切片，保持原有元素顺序。
//
// 示例：
//
//	users := []User{{ID: 1}, {ID: 2}, {ID: 1}}
//	unique := RemoveDuplicates(users, func(u User) int { return u.ID })
//	// unique = [{ID: 1}, {ID: 2}]
func RemoveDuplicates[T any, K comparable](slice []T, keyFunc func(T) K) []T {
	encountered := make(map[K]bool)
	var result []T

	for _, v := range slice {
		key := keyFunc(v)
		if !encountered[key] {
			encountered[key] = true
			result = append(result, v)
		}
	}
	return result
}
