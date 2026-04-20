package models

// User represents a Telegram user or bot.
type User struct {
	ID                      int64  `json:"id"`
	IsBot                   bool   `json:"is_bot"`
	FirstName               string `json:"first_name"`
	LastName                string `json:"last_name,omitempty"`
	Username                string `json:"username,omitempty"`
	LanguageCode            string `json:"language_code,omitempty"`
	IsPremium               bool   `json:"is_premium,omitempty"`
	AddedToAttachmentMenu   bool   `json:"added_to_attachment_menu,omitempty"`
	CanJoinGroups           bool   `json:"can_join_groups,omitempty"`
	CanReadAllGroupMessages bool   `json:"can_read_all_group_messages,omitempty"`
	SupportsInlineQueries   bool   `json:"supports_inline_queries,omitempty"`
	CanConnectToBusiness    bool   `json:"can_connect_to_business,omitempty"`
	HasMainWebApp           bool   `json:"has_main_web_app,omitempty"`
}

// Chat represents a Telegram chat.
type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"` // "private", "group", "supergroup", "channel"
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	IsForum   bool   `json:"is_forum,omitempty"`
}

// ChatFullInfo contains full information about a chat (returned by getChat).
type ChatFullInfo struct {
	Chat
	Description            string           `json:"description,omitempty"`
	InviteLink             string           `json:"invite_link,omitempty"`
	PinnedMessage          *Message         `json:"pinned_message,omitempty"`
	Permissions            *ChatPermissions `json:"permissions,omitempty"`
	SlowModeDelay          int              `json:"slow_mode_delay,omitempty"`
	MessageAutoDeleteTime  int              `json:"message_auto_delete_time,omitempty"`
	HasProtectedContent    bool             `json:"has_protected_content,omitempty"`
	HasVisibleHistory      bool             `json:"has_visible_history,omitempty"`
	StickerSetName         string           `json:"sticker_set_name,omitempty"`
	CanSetStickerSet       bool             `json:"can_set_sticker_set,omitempty"`
	LinkedChatID           int64            `json:"linked_chat_id,omitempty"`
	MaxReactionCount       int              `json:"max_reaction_count,omitempty"`
	AccentColorID          int              `json:"accent_color_id,omitempty"`
}

// Message represents a Telegram message.
type Message struct {
	MessageID            int                  `json:"message_id"`
	MessageThreadID      int                  `json:"message_thread_id,omitempty"`
	From                 *User                `json:"from,omitempty"`
	SenderChat           *Chat                `json:"sender_chat,omitempty"`
	Date                 int64                `json:"date"`
	Chat                 Chat                 `json:"chat"`
	ForwardOrigin        *MessageOrigin       `json:"forward_origin,omitempty"`
	IsAutomaticForward   bool                 `json:"is_automatic_forward,omitempty"`
	ReplyToMessage       *Message             `json:"reply_to_message,omitempty"`
	ExternalReply        *ExternalReplyInfo   `json:"external_reply,omitempty"`
	Quote                *TextQuote           `json:"quote,omitempty"`
	ReplyToStory         *Story               `json:"reply_to_story,omitempty"`
	Text                 string               `json:"text,omitempty"`
	Entities             []MessageEntity      `json:"entities,omitempty"`
	LinkPreviewOptions   *LinkPreviewOptions  `json:"link_preview_options,omitempty"`
	Animation            *Animation           `json:"animation,omitempty"`
	Audio                *Audio               `json:"audio,omitempty"`
	Document             *Document            `json:"document,omitempty"`
	Photo                []PhotoSize          `json:"photo,omitempty"`
	Sticker              *Sticker             `json:"sticker,omitempty"`
	Video                *Video               `json:"video,omitempty"`
	Voice                *Voice               `json:"voice,omitempty"`
	VideoNote            *VideoNote           `json:"video_note,omitempty"`
	Caption              string               `json:"caption,omitempty"`
	CaptionEntities      []MessageEntity      `json:"caption_entities,omitempty"`
	HasMediaSpoiler      bool                 `json:"has_media_spoiler,omitempty"`
	Contact              *Contact             `json:"contact,omitempty"`
	Dice                 *Dice                `json:"dice,omitempty"`
	Poll                 *Poll                `json:"poll,omitempty"`
	Venue                *Venue               `json:"venue,omitempty"`
	Location             *Location            `json:"location,omitempty"`
	NewChatMembers       []User               `json:"new_chat_members,omitempty"`
	LeftChatMember       *User                `json:"left_chat_member,omitempty"`
	NewChatTitle         string               `json:"new_chat_title,omitempty"`
	NewChatPhoto         []PhotoSize          `json:"new_chat_photo,omitempty"`
	DeleteChatPhoto      bool                 `json:"delete_chat_photo,omitempty"`
	GroupChatCreated     bool                 `json:"group_chat_created,omitempty"`
	PinnedMessage        *Message             `json:"pinned_message,omitempty"`
	ReplyMarkup          *InlineKeyboard      `json:"reply_markup,omitempty"`
	HasProtectedContent  bool                 `json:"has_protected_content,omitempty"`
	IsTopicMessage       bool                 `json:"is_topic_message,omitempty"`
	// Deprecated: use ForwardOrigin instead. Kept for backward compatibility.
	ForwardFrom *User  `json:"forward_from,omitempty"`
	ForwardDate int64  `json:"forward_date,omitempty"`
}

// MessageOrigin describes the origin of a message.
type MessageOrigin struct {
	Type       string `json:"type"` // "user", "hidden_user", "chat", "channel"
	Date       int64  `json:"date"`
	SenderUser *User  `json:"sender_user,omitempty"`
	SenderChat *Chat  `json:"sender_chat,omitempty"`
}

// ExternalReplyInfo contains information about a message that is being replied to.
type ExternalReplyInfo struct {
	Origin *MessageOrigin `json:"origin,omitempty"`
}

// TextQuote contains information about the quoted part of a message.
type TextQuote struct {
	Text     string          `json:"text"`
	Entities []MessageEntity `json:"entities,omitempty"`
	Position int             `json:"position"`
	IsManual bool            `json:"is_manual,omitempty"`
}

// Story represents a story.
type Story struct {
	Chat Chat  `json:"chat"`
	ID   int   `json:"id"`
}

// MessageEntity represents a special entity in a text message.
type MessageEntity struct {
	Type          string `json:"type"`
	Offset        int    `json:"offset"`
	Length        int    `json:"length"`
	URL           string `json:"url,omitempty"`
	User          *User  `json:"user,omitempty"`
	Language      string `json:"language,omitempty"`
	CustomEmojiID string `json:"custom_emoji_id,omitempty"`
}

// PhotoSize represents one size of a photo.
type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int    `json:"file_size,omitempty"`
}

// Animation represents an animation file (GIF or H.264/MPEG-4 AVC video without sound).
type Animation struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	Duration     int        `json:"duration"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
	FileName     string     `json:"file_name,omitempty"`
	MimeType     string     `json:"mime_type,omitempty"`
	FileSize     int64      `json:"file_size,omitempty"`
}

// Document represents a general file.
type Document struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
	FileName     string     `json:"file_name,omitempty"`
	MimeType     string     `json:"mime_type,omitempty"`
	FileSize     int64      `json:"file_size,omitempty"`
}

// Video represents a video file.
type Video struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	Duration     int        `json:"duration"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
	FileName     string     `json:"file_name,omitempty"`
	MimeType     string     `json:"mime_type,omitempty"`
	FileSize     int64      `json:"file_size,omitempty"`
}

// Audio represents an audio file.
type Audio struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Duration     int        `json:"duration"`
	Performer    string     `json:"performer,omitempty"`
	Title        string     `json:"title,omitempty"`
	MimeType     string     `json:"mime_type,omitempty"`
	FileSize     int64      `json:"file_size,omitempty"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
}

// Voice represents a voice note.
type Voice struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Duration     int    `json:"duration"`
	MimeType     string `json:"mime_type,omitempty"`
	FileSize     int64  `json:"file_size,omitempty"`
}

// VideoNote represents a video message.
type VideoNote struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Length       int        `json:"length"`
	Duration     int        `json:"duration"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
	FileSize     int64      `json:"file_size,omitempty"`
}

// Sticker represents a sticker.
type Sticker struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Type         string     `json:"type"` // "regular", "mask", "custom_emoji"
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	IsAnimated   bool       `json:"is_animated"`
	IsVideo      bool       `json:"is_video"`
	Emoji        string     `json:"emoji,omitempty"`
	SetName      string     `json:"set_name,omitempty"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
	FileSize     int64      `json:"file_size,omitempty"`
}

// Contact represents a phone contact.
type Contact struct {
	PhoneNumber string `json:"phone_number"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name,omitempty"`
	UserID      int64  `json:"user_id,omitempty"`
	VCard       string `json:"vcard,omitempty"`
}

// Location represents a point on the map.
type Location struct {
	Latitude             float64 `json:"latitude"`
	Longitude            float64 `json:"longitude"`
	HorizontalAccuracy   float64 `json:"horizontal_accuracy,omitempty"`
	LivePeriod           int     `json:"live_period,omitempty"`
	Heading              int     `json:"heading,omitempty"`
	ProximityAlertRadius int     `json:"proximity_alert_radius,omitempty"`
}

// Venue represents a venue.
type Venue struct {
	Location        Location `json:"location"`
	Title           string   `json:"title"`
	Address         string   `json:"address"`
	FoursquareID    string   `json:"foursquare_id,omitempty"`
	FoursquareType  string   `json:"foursquare_type,omitempty"`
	GooglePlaceID   string   `json:"google_place_id,omitempty"`
	GooglePlaceType string   `json:"google_place_type,omitempty"`
}

// Poll represents a native poll.
type Poll struct {
	ID                    string          `json:"id"`
	Question              string          `json:"question"`
	QuestionEntities      []MessageEntity `json:"question_entities,omitempty"`
	Options               []PollOption    `json:"options"`
	TotalVoterCount       int             `json:"total_voter_count"`
	IsClosed              bool            `json:"is_closed"`
	IsAnonymous           bool            `json:"is_anonymous"`
	Type                  string          `json:"type"` // "regular" or "quiz"
	AllowsMultipleAnswers bool            `json:"allows_multiple_answers"`
	CorrectOptionID       *int            `json:"correct_option_id,omitempty"`
	Explanation           string          `json:"explanation,omitempty"`
	ExplanationEntities   []MessageEntity `json:"explanation_entities,omitempty"`
	OpenPeriod            int             `json:"open_period,omitempty"`
	CloseDate             int64           `json:"close_date,omitempty"`
}

// PollOption contains information about one answer option in a poll.
type PollOption struct {
	Text         string          `json:"text"`
	TextEntities []MessageEntity `json:"text_entities,omitempty"`
	VoterCount   int             `json:"voter_count"`
}

// Dice represents an animated emoji that displays a random value.
type Dice struct {
	Emoji string `json:"emoji"`
	Value int    `json:"value"`
}

// LinkPreviewOptions describes options for link preview generation.
type LinkPreviewOptions struct {
	IsDisabled       bool   `json:"is_disabled,omitempty"`
	URL              string `json:"url,omitempty"`
	PreferSmallMedia bool   `json:"prefer_small_media,omitempty"`
	PreferLargeMedia bool   `json:"prefer_large_media,omitempty"`
	ShowAboveText    bool   `json:"show_above_text,omitempty"`
}

// ReplyParameters describes reply parameters for the message being sent.
type ReplyParameters struct {
	MessageID                int             `json:"message_id"`
	ChatID                   int64           `json:"chat_id,omitempty"`
	AllowSendingWithoutReply bool            `json:"allow_sending_without_reply,omitempty"`
	Quote                    string          `json:"quote,omitempty"`
	QuoteParseMode           string          `json:"quote_parse_mode,omitempty"`
	QuoteEntities            []MessageEntity `json:"quote_entities,omitempty"`
	QuotePosition            int             `json:"quote_position,omitempty"`
}

// ReactionType describes the type of a reaction.
type ReactionType struct {
	Type          string `json:"type"` // "emoji" or "custom_emoji"
	Emoji         string `json:"emoji,omitempty"`
	CustomEmojiID string `json:"custom_emoji_id,omitempty"`
}

// WebAppInfo describes a Web App.
type WebAppInfo struct {
	URL string `json:"url"`
}

// LoginUrl represents a parameter of the inline keyboard button used to
// automatically authorize a user.
type LoginUrl struct {
	URL                string `json:"url"`
	ForwardText        string `json:"forward_text,omitempty"`
	BotUsername        string `json:"bot_username,omitempty"`
	RequestWriteAccess bool   `json:"request_write_access,omitempty"`
}

// Update represents an incoming update from Telegram.
type Update struct {
	UpdateID           int64                `json:"update_id"`
	Message            *Message             `json:"message,omitempty"`
	EditedMessage      *Message             `json:"edited_message,omitempty"`
	ChannelPost        *Message             `json:"channel_post,omitempty"`
	EditedChannelPost  *Message             `json:"edited_channel_post,omitempty"`
	CallbackQuery      *CallbackQuery       `json:"callback_query,omitempty"`
	MessageReaction    *MessageReactionUpdated `json:"message_reaction,omitempty"`
	ChatMember         *ChatMemberUpdated   `json:"chat_member,omitempty"`
	MyChatMember       *ChatMemberUpdated   `json:"my_chat_member,omitempty"`
	ChatJoinRequest    *ChatJoinRequest     `json:"chat_join_request,omitempty"`
}

// CallbackQuery represents a callback query from an inline keyboard button.
type CallbackQuery struct {
	ID              string   `json:"id"`
	From            User     `json:"from"`
	Message         *Message `json:"message,omitempty"`
	InlineMessageID string   `json:"inline_message_id,omitempty"`
	ChatInstance    string   `json:"chat_instance"`
	Data            string   `json:"data,omitempty"`
	GameShortName   string   `json:"game_short_name,omitempty"`
}

// MessageReactionUpdated represents a change of a reaction on a message.
type MessageReactionUpdated struct {
	Chat        Chat           `json:"chat"`
	MessageID   int            `json:"message_id"`
	User        *User          `json:"user,omitempty"`
	ActorChat   *Chat          `json:"actor_chat,omitempty"`
	Date        int64          `json:"date"`
	OldReaction []ReactionType `json:"old_reaction"`
	NewReaction []ReactionType `json:"new_reaction"`
}

// ChatJoinRequest represents a join request sent to a chat.
type ChatJoinRequest struct {
	Chat       Chat   `json:"chat"`
	From       User   `json:"from"`
	UserChatID int64  `json:"user_chat_id"`
	Date       int64  `json:"date"`
	Bio        string `json:"bio,omitempty"`
	InviteLink *ChatInviteLink `json:"invite_link,omitempty"`
}

// ChatInviteLink represents an invite link for a chat.
type ChatInviteLink struct {
	InviteLink              string `json:"invite_link"`
	Creator                 User   `json:"creator"`
	CreatesJoinRequest      bool   `json:"creates_join_request"`
	IsPrimary               bool   `json:"is_primary"`
	IsRevoked               bool   `json:"is_revoked"`
	Name                    string `json:"name,omitempty"`
	ExpireDate              int64  `json:"expire_date,omitempty"`
	MemberLimit             int    `json:"member_limit,omitempty"`
	PendingJoinRequestCount int    `json:"pending_join_request_count,omitempty"`
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
	CanSendAudios         bool `json:"can_send_audios,omitempty"`
	CanSendDocuments      bool `json:"can_send_documents,omitempty"`
	CanSendPhotos         bool `json:"can_send_photos,omitempty"`
	CanSendVideos         bool `json:"can_send_videos,omitempty"`
	CanSendVideoNotes     bool `json:"can_send_video_notes,omitempty"`
	CanSendVoiceNotes     bool `json:"can_send_voice_notes,omitempty"`
	CanSendPolls          bool `json:"can_send_polls,omitempty"`
	CanSendOtherMessages  bool `json:"can_send_other_messages,omitempty"`
	CanAddWebPagePreviews bool `json:"can_add_web_page_previews,omitempty"`
	CanChangeInfo         bool `json:"can_change_info,omitempty"`
	CanInviteUsers        bool `json:"can_invite_users,omitempty"`
	CanPinMessages        bool `json:"can_pin_messages,omitempty"`
	CanManageTopics       bool `json:"can_manage_topics,omitempty"`
}

// InlineKeyboard represents an inline keyboard.
type InlineKeyboard struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// InlineKeyboardButton represents one button of an inline keyboard.
type InlineKeyboardButton struct {
	Text                         string      `json:"text"`
	URL                          string      `json:"url,omitempty"`
	CallbackData                 string      `json:"callback_data,omitempty"`
	WebApp                       *WebAppInfo `json:"web_app,omitempty"`
	LoginURL                     *LoginUrl   `json:"login_url,omitempty"`
	SwitchInlineQuery            string      `json:"switch_inline_query,omitempty"`
	SwitchInlineQueryCurrentChat string      `json:"switch_inline_query_current_chat,omitempty"`
	Pay                          bool        `json:"pay,omitempty"`
}

// BotCommand represents a bot command.
type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// File represents a file ready to be downloaded.
type File struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileSize     int64  `json:"file_size,omitempty"`
	FilePath     string `json:"file_path,omitempty"`
}

// WebhookInfo contains information about the webhook.
type WebhookInfo struct {
	URL                          string   `json:"url"`
	HasCustomCertificate         bool     `json:"has_custom_certificate"`
	PendingUpdateCount           int      `json:"pending_update_count"`
	IPAddress                    string   `json:"ip_address,omitempty"`
	LastErrorDate                int64    `json:"last_error_date,omitempty"`
	LastErrorMessage             string   `json:"last_error_message,omitempty"`
	LastSynchronizationErrorDate int64    `json:"last_synchronization_error_date,omitempty"`
	MaxConnections               int      `json:"max_connections,omitempty"`
	AllowedUpdates               []string `json:"allowed_updates,omitempty"`
}

// UserProfilePhotos represents a user's profile pictures.
type UserProfilePhotos struct {
	TotalCount int           `json:"total_count"`
	Photos     [][]PhotoSize `json:"photos"`
}
