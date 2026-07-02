package models

import "encoding/json"

// RichText is a union type for all rich text elements (RichTextBold, RichTextItalic, etc.).
// Stored as raw JSON to accept any variant without exhaustive type definitions.
type RichText = json.RawMessage

// RichBlock is a union type for all rich block types (RichBlockParagraph, RichBlockList, etc.).
type RichBlock = json.RawMessage

// RichBlockCaption represents the caption of a rich message.
type RichBlockCaption struct {
	Text RichText `json:"text"`
}

// RichMessage represents a rich formatted message returned by the API.
type RichMessage struct {
	Caption *RichBlockCaption `json:"caption,omitempty"`
	Blocks  []RichBlock       `json:"blocks"`
}

// InputRichMessage describes a rich message to send via sendRichMessage.
type InputRichMessage struct {
	Caption *RichBlockCaption `json:"caption,omitempty"`
	Blocks  json.RawMessage   `json:"blocks"` // array of RichBlock objects
}

// InputRichMessageContent can be used as InputMessageContent in inline/guest/webapp query results.
type InputRichMessageContent struct {
	MessageText string            `json:"message_text"`
	Blocks      json.RawMessage   `json:"blocks"`
	Caption     *RichBlockCaption `json:"caption,omitempty"`
}
