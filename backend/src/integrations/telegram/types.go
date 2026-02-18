package telegram

type chatMemberStatus = string

const (
	Member chatMemberStatus = "member"
	Kicked chatMemberStatus = "kicked"
	// There are more, but we don't need them just yet
)

type chatType = string

const Private chatType = "private"

type chatMemberUpdated struct {
	Chat          *chat       `json:"chat"`
	From          *user       `json:"from"`
	OldChatMember *chatMember `json:"old_chat_member"`
	NewChatMember *chatMember `json:"new_chat_member"`
}

type chatMember struct {
	User   *user            `json:"user"`
	Status chatMemberStatus `json:"status"`
}

type message struct {
	MessageID int64  `json:"message_id"`
	Date      int64  `json:"date"`
	Text      string `json:"text,omitempty"`
	Chat      chat   `json:"chat"`
	From      *user  `json:"from,omitempty"`
}

type chat struct {
	ID        int64    `json:"id"`
	FirstName string   `json:"first_name"`
	Username  string   `json:"username"`
	Type      chatType `json:"type"`
}

type user struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	IsPremium    bool   `json:"is_premium"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}
