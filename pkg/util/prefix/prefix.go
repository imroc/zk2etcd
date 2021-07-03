package prefix

import "strings"

func Parse(prefix string) []string {
	return strings.Split(prefix, ",")
}

func ShouldExclude(key string, excludePrefixes []string) bool {
	for _, prefix := range excludePrefixes { // exclude prefix
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}
