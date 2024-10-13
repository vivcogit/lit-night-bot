package main

import (
	"fmt"
	"strings"
)

func HandleMultiArgs(rawArgs []string) []string {
	var filtered []string
	for _, str := range rawArgs {
		str = strings.TrimSpace(str)

		if str != "" {
			filtered = append(filtered, str)
		}
	}
	return filtered
}

func GetWishlistMessage(books []string) string {
	var formattedList strings.Builder
	formattedList.WriteString("📚 Ваши книги в вишлисте:\n\n")

	for i, book := range books {
		formattedList.WriteString(fmt.Sprintf("%d. %s\n", i+1, book))
	}

	formattedList.WriteString("\n🎉 Не забудьте выбрать книгу для чтения!")

	return formattedList.String()
}
