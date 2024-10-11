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

type WishlistItem struct {
	Book Book `json:"book"`
}

type CurrentBook struct {
	Book Book `json:"book"`
}

type ChatData struct {
	Wishlist []WishlistItem `json:"wishlist"`
	History  []HistoryItem  `json:"history"`
	Current  CurrentBook    `json:"current_book"`
}

func (cd *ChatData) RemoveBookFromWishlist(bookname string) error {
	index := -1
	for i, b := range cd.Wishlist {
		if strings.EqualFold(b.Book.Name, bookname) {
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

func (cd *ChatData) AddBookToWishlist(bookname string) {
	cd.Wishlist = append(cd.Wishlist, WishlistItem{Book{Name: bookname}})
}

func (cd *ChatData) AddBookToHistory(bookname string) {
	cd.History = append(
		cd.History,
		HistoryItem{
			Book: Book{Name: bookname},
			Date: time.Now(),
		},
	)
}