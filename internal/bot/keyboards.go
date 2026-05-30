package bot

import (
	"fmt"

	"basics/internal/storage"
	"github.com/go-telegram/bot/models"
)

// testKeyboard renders the test selection menu, two buttons per row. Each
// button carries the test's database id (test:<id>). User-owned tests are
// marked so they stand out from the curated global tests.
func testKeyboard(tests []storage.Test) *models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton
	for i := 0; i < len(tests); i += 2 {
		row := []models.InlineKeyboardButton{testButton(tests[i])}
		if i+1 < len(tests) {
			row = append(row, testButton(tests[i+1]))
		}
		rows = append(rows, row)
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func testButton(t storage.Test) models.InlineKeyboardButton {
	label := fmt.Sprintf("%s (%d)", t.Title, len(t.Questions))
	if !t.IsGlobal() {
		label = "👤 " + label
	}
	return models.InlineKeyboardButton{
		Text:         label,
		CallbackData: fmt.Sprintf("test:%d", t.ID),
	}
}

// ownedTestsKeyboard renders one row per owned test with Edit and Delete
// buttons (edit:<id> / del:<id>).
func ownedTestsKeyboard(tests []storage.Test) *models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton
	for _, t := range tests {
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: fmt.Sprintf("✏️ %s (%d)", t.Title, len(t.Questions)), CallbackData: fmt.Sprintf("edit:%d", t.ID)},
			{Text: "🗑 Delete", CallbackData: fmt.Sprintf("del:%d", t.ID)},
		})
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// deleteConfirmKeyboard renders the yes/no confirmation for deleting a test.
func deleteConfirmKeyboard(id int64) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "✅ Yes, delete", CallbackData: fmt.Sprintf("delyes:%d", id)},
				{Text: "↩️ Cancel", CallbackData: "delno"},
			},
		},
	}
}

func orderKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "In order", CallbackData: "order:s"},
				{Text: "Shuffle 🔀", CallbackData: "order:r"},
			},
		},
	}
}

func answerKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "A", CallbackData: "ans:0"},
				{Text: "B", CallbackData: "ans:1"},
			},
			{
				{Text: "C", CallbackData: "ans:2"},
				{Text: "D", CallbackData: "ans:3"},
			},
		},
	}
}

func nextKeyboard(last bool) *models.InlineKeyboardMarkup {
	label := "Next ▶"
	if last {
		label = "See results 🏁"
	}
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: label, CallbackData: "next"}},
		},
	}
}

func againKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "Play again 🔄", CallbackData: "again"}},
		},
	}
}
