package state

import (
	"sync"

	"github.com/skrashevich/telegram-mock-ai/internal/models"
)

// Store provides in-memory storage for all Telegram entities.
type Store struct {
	userMu sync.RWMutex
	users  map[int64]*models.User

	chatMu sync.RWMutex
	chats  map[int64]*models.Chat

	msgMu    sync.RWMutex
	messages map[int64]map[int]*models.Message // chatID -> messageID -> message

	memberMu sync.RWMutex
	members  map[int64]map[int64]*models.ChatMember // chatID -> userID -> member

	userIDGen *models.IDGenerator
	chatIDGen *models.IDGenerator
	msgIDGens map[int64]*models.MessageIDGenerator
	msgIDMu   sync.Mutex
}

// NewStore creates a new in-memory state store.
func NewStore() *Store {
	return &Store{
		users:     make(map[int64]*models.User),
		chats:     make(map[int64]*models.Chat),
		messages:  make(map[int64]map[int]*models.Message),
		members:   make(map[int64]map[int64]*models.ChatMember),
		userIDGen: models.NewIDGenerator(1000),
		chatIDGen: models.NewIDGenerator(0),
		msgIDGens: make(map[int64]*models.MessageIDGenerator),
	}
}

// --- Users ---

// CreateUser adds a user to the store. If ID is 0, auto-generates one.
func (s *Store) CreateUser(u models.User) *models.User {
	s.userMu.Lock()
	defer s.userMu.Unlock()
	if u.ID == 0 {
		u.ID = s.userIDGen.Next()
	}
	stored := u
	s.users[u.ID] = &stored
	return &stored
}

// GetUser returns a user by ID.
func (s *Store) GetUser(id int64) (*models.User, bool) {
	s.userMu.RLock()
	defer s.userMu.RUnlock()
	u, ok := s.users[id]
	return u, ok
}

// ListUsers returns all users.
func (s *Store) ListUsers() []models.User {
	s.userMu.RLock()
	defer s.userMu.RUnlock()
	result := make([]models.User, 0, len(s.users))
	for _, u := range s.users {
		result = append(result, *u)
	}
	return result
}

// --- Chats ---

// CreateChat adds a chat to the store. If ID is 0, auto-generates one.
// Group chats get negative IDs.
func (s *Store) CreateChat(c models.Chat) *models.Chat {
	s.chatMu.Lock()
	defer s.chatMu.Unlock()
	if c.ID == 0 {
		id := s.chatIDGen.Next()
		if c.Type != "private" {
			id = -id
		}
		c.ID = id
	}
	stored := c
	s.chats[c.ID] = &stored
	return &stored
}

// GetChat returns a chat by ID.
func (s *Store) GetChat(id int64) (*models.Chat, bool) {
	s.chatMu.RLock()
	defer s.chatMu.RUnlock()
	c, ok := s.chats[id]
	return c, ok
}

// ListChats returns all chats.
func (s *Store) ListChats() []models.Chat {
	s.chatMu.RLock()
	defer s.chatMu.RUnlock()
	result := make([]models.Chat, 0, len(s.chats))
	for _, c := range s.chats {
		result = append(result, *c)
	}
	return result
}

// --- Messages ---

func (s *Store) getOrCreateMsgIDGen(chatID int64) *models.MessageIDGenerator {
	s.msgIDMu.Lock()
	defer s.msgIDMu.Unlock()
	gen, ok := s.msgIDGens[chatID]
	if !ok {
		gen = models.NewMessageIDGenerator(0)
		s.msgIDGens[chatID] = gen
	}
	return gen
}

// StoreMessage stores a message and assigns a message_id if 0.
func (s *Store) StoreMessage(msg models.Message) *models.Message {
	if msg.MessageID == 0 {
		gen := s.getOrCreateMsgIDGen(msg.Chat.ID)
		msg.MessageID = gen.Next()
	}
	s.msgMu.Lock()
	defer s.msgMu.Unlock()
	if s.messages[msg.Chat.ID] == nil {
		s.messages[msg.Chat.ID] = make(map[int]*models.Message)
	}
	stored := msg
	s.messages[msg.Chat.ID][msg.MessageID] = &stored
	return &stored
}

// GetMessage returns a specific message.
func (s *Store) GetMessage(chatID int64, messageID int) (*models.Message, bool) {
	s.msgMu.RLock()
	defer s.msgMu.RUnlock()
	chatMsgs, ok := s.messages[chatID]
	if !ok {
		return nil, false
	}
	m, ok := chatMsgs[messageID]
	return m, ok
}

// GetChatMessages returns the last N messages in a chat, ordered by message_id.
func (s *Store) GetChatMessages(chatID int64, limit int) []models.Message {
	s.msgMu.RLock()
	defer s.msgMu.RUnlock()
	chatMsgs, ok := s.messages[chatID]
	if !ok {
		return nil
	}
	// Collect all message IDs and sort
	all := make([]models.Message, 0, len(chatMsgs))
	for _, m := range chatMsgs {
		all = append(all, *m)
	}
	// Sort by message_id descending, take limit, reverse
	sortByMsgID(all)
	if len(all) > limit {
		all = all[len(all)-limit:]
	}
	return all
}

// DeleteMessage removes a message from the store.
func (s *Store) DeleteMessage(chatID int64, messageID int) bool {
	s.msgMu.Lock()
	defer s.msgMu.Unlock()
	chatMsgs, ok := s.messages[chatID]
	if !ok {
		return false
	}
	if _, ok := chatMsgs[messageID]; !ok {
		return false
	}
	delete(chatMsgs, messageID)
	return true
}

// --- Chat Members ---

// AddChatMember adds a member to a chat.
func (s *Store) AddChatMember(chatID, userID int64, status string) *models.ChatMember {
	// Look up user before acquiring memberMu to avoid nested locks.
	user, _ := s.GetUser(userID)
	if user == nil {
		return nil
	}
	s.memberMu.Lock()
	defer s.memberMu.Unlock()
	if s.members[chatID] == nil {
		s.members[chatID] = make(map[int64]*models.ChatMember)
	}
	m := &models.ChatMember{
		User:   *user,
		Status: status,
	}
	s.members[chatID][userID] = m
	return m
}

// GetChatMember returns a specific chat member.
func (s *Store) GetChatMember(chatID, userID int64) (*models.ChatMember, bool) {
	s.memberMu.RLock()
	defer s.memberMu.RUnlock()
	chatMembers, ok := s.members[chatID]
	if !ok {
		return nil, false
	}
	m, ok := chatMembers[userID]
	return m, ok
}

// GetChatMembers returns all members of a chat.
func (s *Store) GetChatMembers(chatID int64) []models.ChatMember {
	s.memberMu.RLock()
	defer s.memberMu.RUnlock()
	chatMembers, ok := s.members[chatID]
	if !ok {
		return nil
	}
	result := make([]models.ChatMember, 0, len(chatMembers))
	for _, m := range chatMembers {
		result = append(result, *m)
	}
	return result
}

// GetChatMemberCount returns the number of members in a chat.
func (s *Store) GetChatMemberCount(chatID int64) int {
	s.memberMu.RLock()
	defer s.memberMu.RUnlock()
	return len(s.members[chatID])
}

// RemoveChatMember removes a member from a chat.
func (s *Store) RemoveChatMember(chatID, userID int64) bool {
	s.memberMu.Lock()
	defer s.memberMu.Unlock()
	chatMembers, ok := s.members[chatID]
	if !ok {
		return false
	}
	if _, ok := chatMembers[userID]; !ok {
		return false
	}
	delete(chatMembers, userID)
	return true
}

// UpdateChatMemberStatus updates a member's status.
func (s *Store) UpdateChatMemberStatus(chatID, userID int64, status string) (*models.ChatMember, bool) {
	s.memberMu.Lock()
	defer s.memberMu.Unlock()
	chatMembers, ok := s.members[chatID]
	if !ok {
		return nil, false
	}
	m, ok := chatMembers[userID]
	if !ok {
		return nil, false
	}
	m.Status = status
	return m, true
}

// GetUserChats returns all chats a user is a member of.
func (s *Store) GetUserChats(userID int64) []models.Chat {
	// Collect chat IDs under memberMu, then release before acquiring chatMu
	// to avoid nested lock ordering issues.
	s.memberMu.RLock()
	var chatIDs []int64
	for chatID, members := range s.members {
		if _, ok := members[userID]; ok {
			chatIDs = append(chatIDs, chatID)
		}
	}
	s.memberMu.RUnlock()

	s.chatMu.RLock()
	defer s.chatMu.RUnlock()
	result := make([]models.Chat, 0, len(chatIDs))
	for _, id := range chatIDs {
		if c, ok := s.chats[id]; ok {
			result = append(result, *c)
		}
	}
	return result
}

// GetNonBotChatMembers returns non-bot members of a chat.
func (s *Store) GetNonBotChatMembers(chatID int64) []models.ChatMember {
	s.memberMu.RLock()
	members := make([]models.ChatMember, 0)
	chatMembers, ok := s.members[chatID]
	if ok {
		for _, m := range chatMembers {
			if !m.User.IsBot {
				members = append(members, *m)
			}
		}
	}
	s.memberMu.RUnlock()
	return members
}

// UpdateMessageText updates a message's text under the write lock.
func (s *Store) UpdateMessageText(chatID int64, messageID int, text string, replyMarkup *models.InlineKeyboard) (*models.Message, bool) {
	s.msgMu.Lock()
	defer s.msgMu.Unlock()
	chatMsgs, ok := s.messages[chatID]
	if !ok {
		return nil, false
	}
	m, ok := chatMsgs[messageID]
	if !ok {
		return nil, false
	}
	m.Text = text
	if replyMarkup != nil {
		m.ReplyMarkup = replyMarkup
	}
	copy := *m
	return &copy, true
}

// UpdateMessageReplyMarkup updates a message's reply markup under the write lock.
func (s *Store) UpdateMessageReplyMarkup(chatID int64, messageID int, replyMarkup *models.InlineKeyboard) (*models.Message, bool) {
	s.msgMu.Lock()
	defer s.msgMu.Unlock()
	chatMsgs, ok := s.messages[chatID]
	if !ok {
		return nil, false
	}
	m, ok := chatMsgs[messageID]
	if !ok {
		return nil, false
	}
	m.ReplyMarkup = replyMarkup
	copy := *m
	return &copy, true
}

// sortByMsgID sorts messages by message_id ascending.
func sortByMsgID(msgs []models.Message) {
	for i := 1; i < len(msgs); i++ {
		for j := i; j > 0 && msgs[j].MessageID < msgs[j-1].MessageID; j-- {
			msgs[j], msgs[j-1] = msgs[j-1], msgs[j]
		}
	}
}
