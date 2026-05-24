package broker

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

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

	// ВАЖНО: autoAck теперь false! Мы сами скажем RabbitMQ, когда удалять сообщение
	msgs, err := ch.Consume(qName, "", false, false, false, false, nil)
	if err != nil {
		log.Printf("Ошибка Consumer: %v", err)
		return
	}

	log.Printf("🎧 Слушатель запущен для отдела %s", qName)

	for d := range msgs {
		var ticket TicketMessage
		err := json.Unmarshal(d.Body, &ticket)
		if err != nil {
			d.Ack(false) // Если сообщение битое, удаляем его, чтобы не засорять очередь
			continue
		}

		operators := repository.GetOnlineOperators(db, ticket.DepartmentID)

		// ЛОГИКА: Если нет свободных операторов
		if len(operators) == 0 {
			// Возвращаем в очередь
			d.Nack(false, true)
			// ВАЖНО: Пауза должна быть больше, чем время обновления статуса в БД
			time.Sleep(5 * time.Second)
			continue
		}

		// Если операторы ЕСТЬ, мы берем только ОДНОГО из них (первого свободного)
		// чтобы не спамить всех подряд.
		targetOpID := operators[0]

		text := fmt.Sprintf("🔔 НОВЫЙ ЗАПРОС!\nКлиент ID: %d ожидает ответа.", ticket.UserID)
		acceptButton := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Принять чат", fmt.Sprintf("accept_%d", ticket.UserID)),
			),
		)

		msg := tgbotapi.NewMessage(targetOpID, text)
		msg.ReplyMarkup = acceptButton

		_, err = bot.Send(msg)
		if err != nil {
			// Если бот не смог отправить сообщение оператору (например, он заблокировал бота),
			// возвращаем заявку в очередь
			d.Nack(false, true)
		} else {
			// Успешно отправили — удаляем из очереди
			d.Ack(false)
		}
	}
}
