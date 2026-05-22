package repository

import (
	"database/sql"
	"log"
)

// StartChat связывает клиента и оператора
func StartChat(db *sql.DB, clientID int64, operatorTelegramID int64) error {
	tx, err := db.Begin()
	if err != nil {
		log.Println("Ошибка старта транзакции:", err)
		return err
	}

	// 1. Создаем сессию чата со статусом ACTIVE.
	// Здесь мы просим БД: "Найди внутренний id оператора, у которого telegram_id совпадает с переданным"
	_, err = tx.Exec(`
		INSERT INTO chat_sessions (client_id, operator_id, department_id, status) 
		VALUES (
			$1, 
			(SELECT id FROM operators WHERE telegram_id = $2), 
			(SELECT department_id FROM operators WHERE telegram_id = $2), 
			'ACTIVE'
		)`,
		clientID, operatorTelegramID)

	if err != nil {
		log.Println("Ошибка базы при связывании чата:", err)
		tx.Rollback() // Отменяем транзакцию при ошибке
		return err
	}

	// 2. Меняем статус клиента
	_, err = tx.Exec(`UPDATE users SET bot_state = 'IN_CHAT' WHERE id = $1`, clientID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 3. Меняем статус оператора
	_, err = tx.Exec(`UPDATE operators SET status = 'BUSY' WHERE telegram_id = $1`, operatorTelegramID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Сохраняем изменения навсегда
	return tx.Commit()
}

// GetPartnerID находит Telegram ID собеседника для пересылки сообщений
func GetPartnerID(db *sql.DB, userID int64) int64 {
	var partnerID int64

	// Сценарий А: Пишет КЛИЕНТ. Нам нужен telegram_id оператора (используем JOIN для объединения таблиц).
	queryClient := `
		SELECT o.telegram_id 
		FROM chat_sessions cs 
		JOIN operators o ON cs.operator_id = o.id 
		WHERE cs.client_id = $1 AND cs.status = 'ACTIVE'
	`
	err := db.QueryRow(queryClient, userID).Scan(&partnerID)
	if err == nil {
		return partnerID
	}

	// Сценарий Б: Пишет ОПЕРАТОР. Нам нужен telegram_id клиента.
	queryOperator := `
		SELECT cs.client_id 
		FROM chat_sessions cs 
		JOIN operators o ON cs.operator_id = o.id 
		WHERE o.telegram_id = $1 AND cs.status = 'ACTIVE'
	`
	err = db.QueryRow(queryOperator, userID).Scan(&partnerID)
	if err == nil {
		return partnerID
	}

	return 0 // Человек ни с кем не общается
}

// EndChat закрывает сессию и освобождает обоих участников.
// Возвращает ID собеседника, чтобы мы могли отправить ему уведомление.
func EndChat(db *sql.DB, initiatorID int64) (int64, error) {
	// 1. Узнаем, с кем общался инициатор завершения
	partnerID := GetPartnerID(db, initiatorID)
	if partnerID == 0 {
		return 0, nil // Активного чата нет
	}

	tx, err := db.Begin()
	if err != nil {
		log.Println("Ошибка старта транзакции при завершении:", err)
		return 0, err
	}

	// 2. Закрываем саму сессию чата (фиксируем время окончания)
	_, err = tx.Exec(`
		UPDATE chat_sessions 
		SET status = 'CLOSED', closed_at = CURRENT_TIMESTAMP 
		WHERE status = 'ACTIVE' AND 
		(client_id = $1 OR operator_id = (SELECT id FROM operators WHERE telegram_id = $1))
	`, initiatorID)

	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// 3. Возвращаем клиента в главное меню (кто бы из них двоих ни был клиентом)
	_, err = tx.Exec(`UPDATE users SET bot_state = 'MAIN_MENU' WHERE id = $1 OR id = $2`, initiatorID, partnerID)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// 4. Возвращаем оператора в статус ONLINE, чтобы он мог принимать новые заявки
	_, err = tx.Exec(`UPDATE operators SET status = 'ONLINE' WHERE telegram_id = $1 OR telegram_id = $2`, initiatorID, partnerID)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	err = tx.Commit()
	return partnerID, err
}
