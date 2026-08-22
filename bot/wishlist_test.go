package bot

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestWishlistForceReplyTargetsCallbackUser(t *testing.T) {
	user := &tgbotapi.User{ID: 42, FirstName: "Анна", LastName: "Книги"}
	request := wishlistAddRequestConfig(-100123, user)

	if len(request.Entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(request.Entities))
	}
	entity := request.Entities[0]
	if entity.Type != "text_mention" || entity.User == nil || entity.User.ID != user.ID {
		t.Fatalf("unexpected mention entity: %#v", entity)
	}
	if entity.Offset != 0 || entity.Length != 10 {
		t.Fatalf("unexpected UTF-16 range: offset=%d length=%d", entity.Offset, entity.Length)
	}
	forceReply, ok := request.ReplyMarkup.(tgbotapi.ForceReply)
	if !ok {
		t.Fatalf("reply markup type = %T", request.ReplyMarkup)
	}
	if !forceReply.ForceReply || !forceReply.Selective {
		t.Fatalf("unexpected ForceReply: %#v", forceReply)
	}
}

func TestWishlistForceReplyHandlesUTF16MentionLength(t *testing.T) {
	user := &tgbotapi.User{ID: 43, FirstName: "📚 Анна"}
	request := wishlistAddRequestConfig(-100123, user)
	if got, want := request.Entities[0].Length, 7; got != want {
		t.Fatalf("UTF-16 length = %d, want %d", got, want)
	}
}
