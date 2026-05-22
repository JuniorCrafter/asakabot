package broker

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"asakabot/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	amqp "github.com/rabbitmq/amqp091-go"
)

// StartConsumer теперь принимает доступ к боту и базе данных
func StartConsumer(conn *amqp.Connection, deptID string, bot *tgbotapi.BotAPI, db *sql.DB) {
	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Ошибка канала: %v", err)
		return
	}

	qName := "support_queue_" + deptID

	// НОВОЕ: Обязательно объявляем очередь перед подпиской!
	// Это гарантирует, что слушатель не упадет, если очередь еще пуста
	_, err = ch.QueueDeclare(
		qName,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	)
	if err != nil {
		log.Printf("Ошибка при создании очереди для Consumer %s: %v", qName, err)
		return
	}

	msgs, err := ch.Consume(qName, "", true, false, false, false, nil)
	if err != nil {
		log.Printf("Ошибка Consumer: %v", err)
		return
	}

	log.Printf("🎧 Слушатель запущен для отдела %s", qName)

	for d := range msgs {
		var ticket TicketMessage
		json.Unmarshal(d.Body, &ticket)

		// 1. Находим всех операторов этого отдела, которые сейчас ONLINE
		operators := repository.GetOnlineOperators(db, ticket.DepartmentID)

		if len(operators) == 0 {
			log.Printf("⚠️ Нет доступных операторов для отдела %s", ticket.DepartmentID)
			continue
		}

		// 2. Формируем сообщение для оператора с кнопкой "Принять"
		text := fmt.Sprintf("🔔 НОВЫЙ ЗАПРОС!\nКлиент ID: %d ожидает ответа.", ticket.UserID)
		acceptButton := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Принять чат", fmt.Sprintf("accept_%d", ticket.UserID)),
			),
		)

		// 3. Рассылаем это уведомление всем найденным операторам
		for _, opID := range operators {
			msg := tgbotapi.NewMessage(opID, text)
			msg.ReplyMarkup = acceptButton

			_, err := bot.Send(msg)
			if err != nil {
				log.Printf("Не удалось отправить тикет оператору %d: %v", opID, err)
			}
		}
	}
}
