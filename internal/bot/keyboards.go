package bot

import (
	"fmt"

	"basics/internal/quiz"
	"github.com/go-telegram/bot/models"
)

func categoryKeyboard(store interface {
	All() []quiz.Topic
}) *models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton
	all := store.All()
	for i := 0; i < len(quiz.CategoryMenu); i += 2 {
		e1 := quiz.CategoryMenu[i]
		btn1 := models.InlineKeyboardButton{
			Text:         fmt.Sprintf("%s (%d)", e1.Label, quiz.CountInCategory(all, e1.Cat)),
			CallbackData: fmt.Sprintf("cat:%c", e1.Key),
		}
		row := []models.InlineKeyboardButton{btn1}
		if i+1 < len(quiz.CategoryMenu) {
			e2 := quiz.CategoryMenu[i+1]
			btn2 := models.InlineKeyboardButton{
				Text:         fmt.Sprintf("%s (%d)", e2.Label, quiz.CountInCategory(all, e2.Cat)),
				CallbackData: fmt.Sprintf("cat:%c", e2.Key),
			}
			row = append(row, btn2)
		}
		rows = append(rows, row)
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
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
