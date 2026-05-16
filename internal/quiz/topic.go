package quiz

// Topic represents a single learning item from the CS Foundations Pyramid.
type Topic struct {
	Category    string `json:"Category"`
	Layer       string `json:"Layer"`
	Name        string `json:"Name"`
	Overview    string `json:"Overview"`
	Question    string `json:"Question"`
	Explanation string `json:"Explanation"`
}
