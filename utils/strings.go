package utils

import (
	"strings"
)

func CleanStrSlice(rawArgs []string) []string {
	var filtered []string
	for _, str := range rawArgs {
		str = strings.TrimSpace(str)

		if str != "" {
			filtered = append(filtered, str)
		}
	}
	return filtered
}
