package repository

import (
	"database/sql"
	"fmt"
	"time"
)

// OperatorInfo хранит сводку о сотруднике
type OperatorInfo struct {
	Name       string
	DeptID     int // НОВОЕ ПОЛЕ: номер отдела для перевода
	DeptName   string
	Status     string
	HasSession bool
}

// GetOperatorInfo собирает статистику для профиля
func GetOperatorInfo(db *sql.DB, tgID int64) OperatorInfo {
	var info OperatorInfo

	// ОБНОВЛЕНО: Сканируем department_id сразу в info.DeptID
	err := db.QueryRow(`SELECT name, department_id, status FROM operators WHERE telegram_id = $1`, tgID).Scan(&info.Name, &info.DeptID, &info.Status)
	if err != nil {
		return info
	}

	// Название отдела (перевод ID в текст по умолчанию для самого оператора)
	switch info.DeptID {
	case 1:
		info.DeptName = "Mahalla bankirlari"
	case 2:
		info.DeptName = "Yuridik xizmatlar"
	case 3:
		info.DeptName = "Chakana xizmat"
	case 4:
		info.DeptName = "Общие вопросы"
	default:
		info.DeptName = "Неизвестно"
	}

	// Проверяем, есть ли сейчас активный чат с клиентом
	var count int
	db.QueryRow(`SELECT count(*) FROM chat_sessions WHERE operator_id = (SELECT id FROM operators WHERE telegram_id = $1) AND status = 'ACTIVE'`, tgID).Scan(&count)
	info.HasSession = count > 0

	return info
}

// SetOperatorStatus меняет статус в базе (ONLINE / OFFLINE)
func SetOperatorStatus(db *sql.DB, tgID int64, status string) error {
	_, err := db.Exec(`UPDATE operators SET status = $1 WHERE telegram_id = $2`, status, tgID)
	return err
}

// Добавляет нового оператора
func AddOperator(db *sql.DB, telegramID int64, name string, deptID int) error {
	_, err := db.Exec(`INSERT INTO operators (telegram_id, name, department_id, status) VALUES ($1, $2, $3, 'OFFLINE')`, telegramID, name, deptID)
	return err
}

// Удаляет оператора
func DeleteOperator(db *sql.DB, telegramID int64) error {
	_, err := db.Exec(`DELETE FROM operators WHERE telegram_id = $1`, telegramID)
	return err
}

// Принудительно меняет статус оператора
func ForceOperatorStatus(db *sql.DB, telegramID int64, status string) error {
	_, err := db.Exec(`UPDATE operators SET status = $1 WHERE telegram_id = $2`, status, telegramID)
	return err
}

// Собирает статистику для панели администратора
func GetSystemStats(db *sql.DB) string {
	var opCount, onlineCount, activeChats int
	db.QueryRow(`SELECT count(*) FROM operators`).Scan(&opCount)
	db.QueryRow(`SELECT count(*) FROM operators WHERE status = 'ONLINE'`).Scan(&onlineCount)
	db.QueryRow(`SELECT count(*) FROM chat_sessions WHERE status = 'ACTIVE'`).Scan(&activeChats)

	return fmt.Sprintf("📊 *Статистика системы:*\n\n👥 Всего операторов: %d\n🟢 Операторов в сети: %d\n💬 Активных диалогов сейчас: %d", opCount, onlineCount, activeChats)
}

func AppendToChatLog(db *sql.DB, userID int64, senderRole, text string) error {
	// Формируем красивую строку: [14:35:00] Оператор: Здравствуйте!
	timestamp := time.Now().Format("15:04:05")
	messageLine := fmt.Sprintf("[%s] %s: %s\n", timestamp, senderRole, text)

	// COALESCE гарантирует, что если ячейка пустая (NULL), ошибки склеивания не будет
	query := `
		UPDATE chat_sessions 
		SET chat_log = COALESCE(chat_log, '') || $1 
		WHERE (client_id = $2 OR operator_id = $2) AND status = 'ACTIVE'
	`
	_, err := db.Exec(query, messageLine, userID)
	return err
}

// UpdateOperatorDept изменяет отдел существующего оператора
func UpdateOperatorDept(db *sql.DB, telegramID int64, newDeptID int) error {
	_, err := db.Exec(`UPDATE operators SET department_id = $1 WHERE telegram_id = $2`, newDeptID, telegramID)
	return err
}
