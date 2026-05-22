package models

// User описывает структуру клиента в базе данных
type User struct {
	ID           int64
	Username     string
	FirstName    string
	LanguageCode string
	BotState     string
}
