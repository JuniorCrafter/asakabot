package broker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// TicketMessage — структура нашего "письма" для оператора
type TicketMessage struct {
	UserID       int64  `json:"user_id"`
	DepartmentID string `json:"department_id"`
	Timestamp    string `json:"timestamp"`
}

// ConnectRabbitMQ устанавливает соединение с брокером
func ConnectRabbitMQ(url string) *amqp.Connection {
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("Ошибка подключения к RabbitMQ: %v", err)
	}
	log.Println("Успешное подключение к RabbitMQ!")
	return conn
}

// PublishTicket отправляет уведомление о новом клиенте в нужную очередь
func PublishTicket(conn *amqp.Connection, userID int64, deptID string) error {
	// Открываем канал связи
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// Название очереди (например, "support_queue_1")
	qName := "support_queue_" + deptID

	// Гарантируем, что очередь существует (создаем, если ее нет)
	_, err = ch.QueueDeclare(
		qName,
		true,  // durable (сохраняется при перезапуске контейнера)
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	)
	if err != nil {
		return err
	}

	// Упаковываем данные клиента в формат JSON
	msg := TicketMessage{
		UserID:       userID,
		DepartmentID: deptID,
		Timestamp:    time.Now().Format(time.RFC3339),
	}
	body, _ := json.Marshal(msg)

	// Тайм-аут на отправку
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Отправляем письмо в очередь
	err = ch.PublishWithContext(ctx,
		"",    // exchange (пока не используем сложную маршрутизацию)
		qName, // routing key (имя очереди)
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})

	if err == nil {
		log.Printf("📥 Тикет от %d отправлен в очередь %s\n", userID, qName)
	}
	return err
}
