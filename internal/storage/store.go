package storage

import "basics/internal/quiz"

// TopicStore is the persistence interface for topic data.
// Implementations may back this with a JSON file, a database, etc.
type TopicStore interface {
	// All returns every topic.
	All() []quiz.Topic
	// ByCategory returns topics whose Category matches cat.
	// An empty cat string returns all topics.
	ByCategory(cat string) []quiz.Topic
}
