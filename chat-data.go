package main

import (
	"fmt"
	"strings"
	"time"
)

type Book struct {
	Name string `json:"name"`
}

type HistoryItem struct {
	Book Book      `json:"book"`
	Date time.Time `json:"date"`
}

type ChatData struct {
	Wishlist []Book        `json:"wishlist"`
	History  []HistoryItem `json:"history"`
	Current  Book          `json:"current_book"`
}

func (cd *ChatData) removeBookFromWishList(bookname string) error {
	index := -1
	for i, b := range cd.Wishlist {
		if strings.EqualFold(b.Name, bookname) {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("книга \"%s\" не найдена в списке", bookname)
	}

	cd.Wishlist = append(cd.Wishlist[:index], cd.Wishlist[index+1:]...)

	return nil
}

func (cd *ChatData) addBookToWishlist(bookname string) {
	cd.Wishlist = append(cd.Wishlist, Book{Name: bookname})
}
