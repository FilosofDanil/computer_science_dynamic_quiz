package bot

import (
	"sync"

	"basics/internal/quiz"
)

// stage represents the current step in a user's quiz flow.
type stage int

const (
	stageCategory stage = iota
	stageOrder
	stageQuiz
	stageReveal
	stageDone
)

// Session holds all per-user state for an in-progress quiz.
type Session struct {
	stage      stage
	topics     []quiz.Topic
	index      int
	score      int
	options    []string
	correctIdx int
	lastMsgID  int
}

// SessionStore manages per-chat sessions.
type SessionStore interface {
	// Get returns the existing session for chatID, creating a new one if absent.
	Get(chatID int64) *Session
	// Reset replaces the session for chatID with a fresh one and returns it.
	Reset(chatID int64) *Session
	// Delete removes the session for chatID.
	Delete(chatID int64)
}

// InMemorySessionStore is a thread-safe, map-backed SessionStore.
type InMemorySessionStore struct {
	mu       sync.Mutex
	sessions map[int64]*Session
}

// NewInMemorySessionStore creates an empty InMemorySessionStore.
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: make(map[int64]*Session),
	}
}

func (s *InMemorySessionStore) Get(chatID int64) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[chatID]
	if !ok {
		sess = &Session{}
		s.sessions[chatID] = sess
	}
	return sess
}

func (s *InMemorySessionStore) Reset(chatID int64) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := &Session{}
	s.sessions[chatID] = sess
	return sess
}

func (s *InMemorySessionStore) Delete(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, chatID)
}
