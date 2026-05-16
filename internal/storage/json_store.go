package storage

import (
	"encoding/json"
	"fmt"
	"os"

	"basics/internal/quiz"
)

// JSONTopicStore loads topics from a JSON file at startup and serves them
// from memory. The file is read exactly once; the store is read-only after
// construction.
type JSONTopicStore struct {
	topics []quiz.Topic
}

// NewJSONTopicStore reads path and unmarshals its contents into a TopicStore.
// Returns an error if the file cannot be read or parsed.
func NewJSONTopicStore(path string) (*JSONTopicStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("storage: read %s: %w", path, err)
	}
	var topics []quiz.Topic
	if err := json.Unmarshal(data, &topics); err != nil {
		return nil, fmt.Errorf("storage: parse %s: %w", path, err)
	}
	if len(topics) == 0 {
		return nil, fmt.Errorf("storage: %s contains no topics", path)
	}
	return &JSONTopicStore{topics: topics}, nil
}

// All returns every topic in the store.
func (s *JSONTopicStore) All() []quiz.Topic {
	out := make([]quiz.Topic, len(s.topics))
	copy(out, s.topics)
	return out
}

// ByCategory returns topics matching cat. An empty string returns all topics.
func (s *JSONTopicStore) ByCategory(cat string) []quiz.Topic {
	return quiz.FilterTopics(s.topics, cat)
}
