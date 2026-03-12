package models

import (
	"crypto/rand"
	"encoding/hex"
	"sync/atomic"
)

// IDGenerator provides monotonically increasing int64 IDs.
type IDGenerator struct {
	counter atomic.Int64
}

// NewIDGenerator creates a generator starting at seed.
func NewIDGenerator(seed int64) *IDGenerator {
	g := &IDGenerator{}
	g.counter.Store(seed)
	return g
}

// Next returns the next ID.
func (g *IDGenerator) Next() int64 {
	return g.counter.Add(1)
}

// MessageIDGenerator provides monotonically increasing int message IDs per chat.
type MessageIDGenerator struct {
	counter atomic.Int32
}

// NewMessageIDGenerator creates a generator starting at seed.
func NewMessageIDGenerator(seed int32) *MessageIDGenerator {
	g := &MessageIDGenerator{}
	g.counter.Store(seed)
	return g
}

// Next returns the next message ID.
func (g *MessageIDGenerator) Next() int {
	return int(g.counter.Add(1))
}

// RandomHex generates a random hex string of n bytes.
func RandomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
