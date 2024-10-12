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

type HasBook interface {
	GetBook() Book
}

func (wi WishlistItem) GetBook() Book {
	return wi.Book
}

func (hi HistoryItem) GetBook() Book {
	return hi.Book
}

func RemoveBookFromBooklist[T HasBook](booklist *[]T, bookname string) error {
	index := -1
	for i, b := range *booklist {
		if strings.EqualFold(b.GetBook().Name, bookname) {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("книга \"%s\" не найдена в списке", bookname)
	}

	*booklist = append((*booklist)[:index], (*booklist)[index+1:]...)

	return nil
}

func GetBooknamesFromBooklist[T HasBook](booklist *[]T) []string {
	var names []string
	for _, item := range *booklist {
		names = append(names, item.GetBook().Name)
	}

	return names
}

func (cd *ChatData) RemoveBookFromWishlist(bookname string) error {
	return RemoveBookFromBooklist(&cd.Wishlist, bookname)
}

func (cd *ChatData) RemoveBookFromHistory(bookname string) error {
	return RemoveBookFromBooklist(&cd.History, bookname)
}

func (cd *ChatData) AddBooksToWishlist(booknames []string) {
	for _, bookname := range booknames {
		cd.AddBookToWishlist(bookname)
	}
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

func (cd *ChatData) AddBooksToHistory(booknames []string) {
	for _, bookname := range booknames {
		cd.AddBookToHistory(bookname)
	}
}

func (cd *ChatData) SetCurrentBook(bookname string) {
	cd.Current = CurrentBook{Book{bookname}}
}

func (cd *ChatData) GetWishlistBooks() []string {
	return GetBooknamesFromBooklist(&cd.Wishlist)
}

func (cd *ChatData) GetHistoryBooks() []string {
	return GetBooknamesFromBooklist(&cd.History)
}
