package main

import "strings"

func TruncateString(str string, maxLength int) string {
	if len(str) > maxLength {
		return str[:maxLength-3] + "..."
	}
	return str
}

func HandleMultiArgs(rawArgs []string) []string {
	var filtered []string
	for _, str := range rawArgs {
		str = TruncateString(strings.TrimSpace(str), 58)

		if str != "" {
			filtered = append(filtered, str)
		}
	}
	return filtered
}
