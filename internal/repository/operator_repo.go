package repository

import (
	"database/sql"
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
