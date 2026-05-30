package apperrors

import "errors"

// Sentinel errors that bot handlers may check with errors.Is.
var (
	// ErrUnknownCategory is returned when a callback contains an unrecognised
	// category key.
	ErrUnknownCategory = errors.New("unknown category key")

	// ErrInvalidStage is returned when a callback arrives in the wrong session
	// stage (e.g. an answer callback while in stageReveal).
	ErrInvalidStage = errors.New("invalid session stage for this action")

	// ErrNoTopics is returned when a category filter produces an empty list.
	ErrNoTopics = errors.New("no topics found for category")

	// ErrTestNotFound is returned when a test lookup, update, or delete finds
	// no matching row (e.g. unknown id, or a delete that the user does not own).
	ErrTestNotFound = errors.New("test not found")

	// ErrInvalidTest is returned when user-supplied test JSON fails validation.
	ErrInvalidTest = errors.New("invalid test")
)
