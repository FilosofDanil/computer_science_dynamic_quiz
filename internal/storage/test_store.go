package storage

import (
	"basics/internal/quiz"
)

// Test is a named set of quiz questions persisted as a single JSON document.
// Curated tests are global (OwnerChat == nil); user-created tests belong to
// the chat that created them (OwnerChat == &chatID).
type Test struct {
	ID        int64
	OwnerChat *int64
	Title     string
	Questions []quiz.Topic
}

// IsGlobal reports whether the test is a curated/global test (no owner).
func (t Test) IsGlobal() bool {
	return t.OwnerChat == nil
}

// OwnedBy reports whether the test belongs to the given chat.
func (t Test) OwnedBy(chatID int64) bool {
	return t.OwnerChat != nil && *t.OwnerChat == chatID
}

// TestStore is the persistence interface for dynamic, user-managed tests.
// It supports both read access (for playing quizzes) and write access (for
// users creating, editing, and deleting their own tests).
type TestStore interface {
	// ListAvailable returns every test the chat can play: all global tests
	// plus the tests owned by chatID, ordered by title.
	ListAvailable(chatID int64) ([]Test, error)
	// ListOwned returns only the tests owned by chatID, ordered by title.
	ListOwned(chatID int64) ([]Test, error)
	// Get returns a single test by id.
	Get(id int64) (Test, error)
	// Create inserts a new test and returns its generated id.
	Create(t Test) (int64, error)
	// Update replaces the title and questions of an existing test. The update
	// only applies to a test owned by t.OwnerChat.
	Update(t Test) error
	// Delete removes the test with the given id, but only if it is owned by
	// ownerChat. Returns ErrTestNotFound if no matching owned test exists.
	Delete(id, ownerChat int64) error
}
