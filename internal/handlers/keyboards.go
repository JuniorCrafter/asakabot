package handlers

import (
	"asakabot/internal/i18n"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func MainMenuKeyboard(lang string) tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(lang, "btn_contacts")),
			tgbotapi.NewKeyboardButton(i18n.Get(lang, "btn_settings")),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(lang, "btn_support")),
			tgbotapi.NewKeyboardButton(i18n.Get(lang, "btn_about")),
		),
	)
	keyboard.ResizeKeyboard = true
	return keyboard
}

// SupportDepartmentsReplyKeyboard создает нижние кнопки отделов с кнопкой Назад (Пункт 2.2.1)
func SupportDepartmentsReplyKeyboard(lang string) tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(lang, "dept_1")),
			tgbotapi.NewKeyboardButton(i18n.Get(lang, "dept_2")),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(lang, "dept_3")),
			tgbotapi.NewKeyboardButton(i18n.Get(lang, "dept_4")),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(lang, "btn_back")),
		),
	)
	keyboard.ResizeKeyboard = true
	return keyboard
}

// RequestPhoneKeyboard создает кнопку отправки контакта
func RequestPhoneKeyboard(lang string) tgbotapi.ReplyKeyboardMarkup {
	btn := tgbotapi.NewKeyboardButton(i18n.Get(lang, "btn_share_phone"))
	btn.RequestContact = true // Запрос контакта у Telegram

	keyboard := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(btn))
	keyboard.ResizeKeyboard = true
	return keyboard
}

// SettingsInlineKeyboard оставляет инлайн-выбор языка (это удобно)
func SettingsInlineKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇷🇺 Русский", "lang_ru"),
			tgbotapi.NewInlineKeyboardButtonData("🇺🇿 O'zbekcha", "lang_uz"),
		),
	)
}

// OperatorMenuKeyboard создает меню в зависимости от текущего статуса
func OperatorMenuKeyboard(status string) tgbotapi.ReplyKeyboardMarkup {
	var statusBtn, toggleBtn tgbotapi.KeyboardButton

	if status == "ONLINE" {
		statusBtn = tgbotapi.NewKeyboardButton("ℹ️ Мой статус: ONLINE")
		toggleBtn = tgbotapi.NewKeyboardButton("🔴 Стать OFFLINE")
	} else {
		statusBtn = tgbotapi.NewKeyboardButton("ℹ️ Мой статус: OFFLINE")
		toggleBtn = tgbotapi.NewKeyboardButton("🟢 Стать ONLINE")
	}

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(statusBtn, toggleBtn),
		// Кнопки "Завершить чат" здесь больше нет!
	)
	keyboard.ResizeKeyboard = true
	return keyboard
}

// OperatorInChatKeyboard показывает только одну кнопку во время диалога
func OperatorInChatKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("❌ Завершить текущий чат"),
		),
	)
	keyboard.ResizeKeyboard = true
	return keyboard
}

func AdminMenuKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("➕ Добавить оператора"),
			tgbotapi.NewKeyboardButton("➖ Удалить оператора"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📊 Статистика"),
			tgbotapi.NewKeyboardButton("🚦 Управление статусами"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔙 Выйти из панели"),
		),
	)
	keyboard.ResizeKeyboard = true
	return keyboard
}
