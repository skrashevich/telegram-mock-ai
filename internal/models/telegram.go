package models

// User represents a Telegram user or bot.
type User struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
	IsPremium    bool   `json:"is_premium,omitempty"`
}

// Chat represents a Telegram chat.
type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"` // "private", "group", "supergroup", "channel"
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// Message represents a Telegram message.
type Message struct {
	MessageID       int              `json:"message_id"`
	From            *User            `json:"from,omitempty"`
	Chat            Chat             `json:"chat"`
	Date            int64            `json:"date"`
	Text            string           `json:"text,omitempty"`
	Entities        []MessageEntity  `json:"entities,omitempty"`
	ReplyToMessage  *Message         `json:"reply_to_message,omitempty"`
	Photo           []PhotoSize      `json:"photo,omitempty"`
	Document        *Document        `json:"document,omitempty"`
	Video           *Video           `json:"video,omitempty"`
	Audio           *Audio           `json:"audio,omitempty"`
	Voice           *Voice           `json:"voice,omitempty"`
	Sticker         *Sticker         `json:"sticker,omitempty"`
	NewChatMembers  []User           `json:"new_chat_members,omitempty"`
	LeftChatMember  *User            `json:"left_chat_member,omitempty"`
	ReplyMarkup     *InlineKeyboard  `json:"reply_markup,omitempty"`
	ForwardFrom     *User            `json:"forward_from,omitempty"`
	ForwardDate     int64            `json:"forward_date,omitempty"`
	PinnedMessage   *Message         `json:"pinned_message,omitempty"`
}

// MessageEntity represents a special entity in a text message.
type MessageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	URL    string `json:"url,omitempty"`
	User   *User  `json:"user,omitempty"`
}

// PhotoSize represents one size of a photo.
type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int    `json:"file_size,omitempty"`
}

// Document represents a general file.
type Document struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileName     string `json:"file_name,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
	FileSize     int64  `json:"file_size,omitempty"`
}

// Video represents a video file.
type Video struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Duration     int    `json:"duration"`
	FileSize     int64  `json:"file_size,omitempty"`
}

// Audio represents an audio file.
type Audio struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Duration     int    `json:"duration"`
	Title        string `json:"title,omitempty"`
	FileSize     int64  `json:"file_size,omitempty"`
}

// Voice represents a voice note.
type Voice struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Duration     int    `json:"duration"`
	FileSize     int64  `json:"file_size,omitempty"`
}

// Sticker represents a sticker.
type Sticker struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Type         string `json:"type"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	IsAnimated   bool   `json:"is_animated"`
	Emoji        string `json:"emoji,omitempty"`
}

// Update represents an incoming update from Telegram.
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	EditedMessage *Message       `json:"edited_message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
	ChatMember    *ChatMemberUpdated `json:"chat_member,omitempty"`
	MyChatMember  *ChatMemberUpdated `json:"my_chat_member,omitempty"`
}

// CallbackQuery represents a callback query from an inline keyboard button.
type CallbackQuery struct {
	ID           string   `json:"id"`
	From         User     `json:"from"`
	Message      *Message `json:"message,omitempty"`
	ChatInstance string   `json:"chat_instance"`
	Data         string   `json:"data,omitempty"`
}

// ChatMember represents a chat member with status.
type ChatMember struct {
	User   User   `json:"user"`
	Status string `json:"status"` // "creator", "administrator", "member", "restricted", "left", "kicked"
}

// ChatMemberUpdated represents changes in chat member status.
type ChatMemberUpdated struct {
	Chat          Chat       `json:"chat"`
	From          User       `json:"from"`
	Date          int64      `json:"date"`
	OldChatMember ChatMember `json:"old_chat_member"`
	NewChatMember ChatMember `json:"new_chat_member"`
}

// ChatPermissions represents the default permissions of a chat.
type ChatPermissions struct {
	CanSendMessages       bool `json:"can_send_messages,omitempty"`
	CanSendMediaMessages  bool `json:"can_send_media_messages,omitempty"`
	CanSendPolls          bool `json:"can_send_polls,omitempty"`
	CanSendOtherMessages  bool `json:"can_send_other_messages,omitempty"`
	CanAddWebPagePreviews bool `json:"can_add_web_page_previews,omitempty"`
	CanChangeInfo         bool `json:"can_change_info,omitempty"`
	CanInviteUsers        bool `json:"can_invite_users,omitempty"`
	CanPinMessages        bool `json:"can_pin_messages,omitempty"`
}

// InlineKeyboard represents an inline keyboard.
type InlineKeyboard struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// InlineKeyboardButton represents one button of an inline keyboard.
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

// WebhookInfo contains information about the webhook.
type WebhookInfo struct {
	URL                  string   `json:"url"`
	HasCustomCertificate bool     `json:"has_custom_certificate"`
	PendingUpdateCount   int      `json:"pending_update_count"`
	LastErrorDate        int64    `json:"last_error_date,omitempty"`
	LastErrorMessage     string   `json:"last_error_message,omitempty"`
	MaxConnections       int      `json:"max_connections,omitempty"`
	AllowedUpdates       []string `json:"allowed_updates,omitempty"`
}
