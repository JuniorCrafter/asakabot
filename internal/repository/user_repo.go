package repository

import (
	"database/sql"
	"log"
)

// SaveUser сохраняет нового пользователя или обновляет существующего
func SaveUser(db *sql.DB, id int64, username string, firstName string) error {
	// SQL-запрос. $1, $2, $3 — это безопасные "заглушки" для данных, защищающие от взлома (SQL-инъекций)
	query := `
		INSERT INTO users (id, username, first_name) 
		VALUES ($1, $2, $3)
		ON CONFLICT (id) 
		DO UPDATE SET 
			username = EXCLUDED.username, 
			first_name = EXCLUDED.first_name;
	`

	// Выполняем запрос
	_, err := db.Exec(query, id, username, firstName)
	if err != nil {
		log.Println("Ошибка при сохранении пользователя:", err)
		return err
	}

	log.Printf("Пользователь %s (ID: %d) сохранен в БД\n", firstName, id)
	return nil
}

// UpdateLanguage обновляет язык пользователя в базе данных
func UpdateLanguage(db *sql.DB, userID int64, langCode string) error {
	query := `UPDATE users SET language_code = $1 WHERE id = $2`

	_, err := db.Exec(query, langCode, userID)
	if err != nil {
		log.Printf("Ошибка при обновлении языка для %d: %v\n", userID, err)
		return err
	}

	log.Printf("Пользователь %d сменил язык на %s\n", userID, langCode)
	return nil
}

// UpdateUserState обновляет текущее состояние пользователя (например, WAITING)
func UpdateUserState(db *sql.DB, userID int64, state string) error {
	query := `UPDATE users SET bot_state = $1 WHERE id = $2`

	_, err := db.Exec(query, state, userID)
	if err != nil {
		log.Printf("Ошибка при обновлении статуса для %d: %v\n", userID, err)
		return err
	}

	return nil
}

// IsOperator проверяет, есть ли пользователь в таблице операторов
func IsOperator(db *sql.DB, telegramID int64) bool {
	var id int
	// Пытаемся найти ID оператора по его Telegram ID
	query := `SELECT id FROM operators WHERE telegram_id = $1`

	err := db.QueryRow(query, telegramID).Scan(&id)
	if err != nil {
		// Если запись не найдена (или другая ошибка), возвращаем false
		return false
	}

	// Если мы дошли сюда, значит человек — оператор
	return true
}

// GetOnlineOperators возвращает список Telegram ID операторов конкретного отдела
func GetOnlineOperators(db *sql.DB, deptID string) []int64 {
	// Ищем только тех, кто в сети
	query := `SELECT telegram_id FROM operators WHERE department_id = $1 AND status = 'ONLINE'`

	rows, err := db.Query(query, deptID)
	if err != nil {
		log.Println("Ошибка при поиске операторов:", err)
		return nil
	}
	defer rows.Close()

	var operators []int64
	for rows.Next() {
		var tgID int64
		if err := rows.Scan(&tgID); err == nil {
			operators = append(operators, tgID)
		}
	}
	return operators
}

// GetUserLang возвращает текущий язык пользователя из БД
func GetUserLang(db *sql.DB, userID int64) string {
	var lang string
	err := db.QueryRow("SELECT language_code FROM users WHERE id = $1", userID).Scan(&lang)
	if err != nil || lang == "" {
		return "ru" // По умолчанию
	}
	return lang
}

// GetUserState возвращает текущее состояние (bot_state) пользователя
func GetUserState(db *sql.DB, userID int64) string {
	var state string
	err := db.QueryRow("SELECT bot_state FROM users WHERE id = $1", userID).Scan(&state)
	if err != nil {
		return ""
	}
	return state
}

// IsUserRegistered проверяет, заполнил ли пользователь профиль
func IsUserRegistered(db *sql.DB, userID int64) bool {
	var realName, phone sql.NullString
	err := db.QueryRow("SELECT real_name, phone FROM users WHERE id = $1", userID).Scan(&realName, &phone)
	if err != nil {
		return false
	}
	return realName.Valid && phone.Valid && realName.String != "" && phone.String != ""
}

// UpdateUserRegistrationName сохраняет реальное имя и переводит на ввод телефона
func UpdateUserRegistrationName(db *sql.DB, userID int64, name string) error {
	_, err := db.Exec("UPDATE users SET real_name = $1, bot_state = 'REG_PHONE' WHERE id = $2", name, userID)
	return err
}

// UpdateUserRegistrationPhone сохраняет телефон и переводит в главное меню
func UpdateUserRegistrationPhone(db *sql.DB, userID int64, phone string) error {
	_, err := db.Exec("UPDATE users SET phone = $1, bot_state = 'MAIN_MENU' WHERE id = $2", phone, userID)
	return err
}

// Обновляет имя пользователя после ввода
func UpdateUserName(db *sql.DB, telegramID int64, name string) error {
	_, err := db.Exec(`UPDATE users SET first_name = $1 WHERE telegram_id = $2`, name, telegramID)
	return err
}
