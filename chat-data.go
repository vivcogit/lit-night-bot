package main

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Book struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

func NewBook(name string) Book {
	return Book{
		name,
		getUuid(),
	}
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

func getUuid() string {
	return uuid.New().String()[:8]
}

func (wi WishlistItem) GetBook() Book {
	return wi.Book
}

func (hi HistoryItem) GetBook() Book {
	return hi.Book
}

func RemoveBookFromBooklist[T HasBook](booklist *[]T, UUID string) (*Book, error) {
	index := -1
	for i, b := range *booklist {
		if strings.EqualFold(b.GetBook().UUID, UUID) {
			index = i
			break
		}
	}

	if index == -1 {
		return nil, errors.New("книга не найдена в списке")
	}

	book := (*booklist)[index].GetBook()
	*booklist = append((*booklist)[:index], (*booklist)[index+1:]...)

	return &book, nil
}

func GetBooknamesFromBooklist[T HasBook](booklist *[]T) []string {
	var names []string
	for _, item := range *booklist {
		names = append(names, item.GetBook().Name)
	}

	return names
}

func (cd *ChatData) RemoveBookFromWishlist(uuid string) (*Book, error) {
	return RemoveBookFromBooklist(&cd.Wishlist, uuid)
}

func (cd *ChatData) RemoveBookFromHistory(uuid string) (*Book, error) {
	return RemoveBookFromBooklist(&cd.History, uuid)
}

func (cd *ChatData) AddBooksToWishlist(booknames []string) {
	for _, bookname := range booknames {
		cd.AddBookToWishlist(bookname)
	}
}

func (cd *ChatData) AddBookToWishlist(bookname string) {
	cd.Wishlist = append(cd.Wishlist, WishlistItem{NewBook(bookname)})
}

func (cd *ChatData) AddBookToHistory(bookname string) {
	cd.History = append(
		cd.History,
		HistoryItem{
			Book: NewBook(bookname),
			Date: time.Now(),
		},
	)
}

func (cd *ChatData) AddBooksToHistory(booknames []string) {
	for _, bookname := range booknames {
		cd.AddBookToHistory(bookname)
	}
}

func (cd *ChatData) SetCurrentBook(book Book) {
	cd.Current = CurrentBook{book}
}

func (cd *ChatData) GetWishlistBooks() []string {
	return GetBooknamesFromBooklist(&cd.Wishlist)
}

func (cd *ChatData) GetHistoryBooks() []string {
	return GetBooknamesFromBooklist(&cd.History)
}
