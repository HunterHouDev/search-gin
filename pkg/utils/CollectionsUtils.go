package utils

import (
	"strings"
)

// HasItem 判断集合是否包含
func HasItem(lib []string, item string) bool {
	if lib == nil {
		return false
	}
	if len(lib) == 0 {
		return false
	}
	for i := 0; i < len(lib); i++ {
		if strings.Compare(item, lib[i]) == 0 {
			return true
		}
	}
	return false
}

func HasItemSet(set map[string]struct{}, item string) bool {
	_, ok := set[item]
	return ok
}

func ToSet(lib []string) map[string]struct{} {
	set := make(map[string]struct{}, len(lib))
	for _, item := range lib {
		set[item] = struct{}{}
	}
	return set
}

func ExtendsItems[T any](lib []T, items []T) []T {
	lib = append(lib, items...)
	return lib
}

func IndexOf(lib []string, item string) int {
	if lib == nil {
		return -1
	}
	if len(lib) == 0 {
		return -1
	}
	for i := 0; i < len(lib); i++ {

		if strings.Compare(item, lib[i]) == 0 {
			return i
		}
	}
	return -1
}
