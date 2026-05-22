package repository

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq" // Импортируем драйвер postgres (знак _ означает, что он работает в фоне)
)

// ConnectDB открывает соединение с PostgreSQL и проверяет его
func ConnectDB(dbUrl string) *sql.DB {
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal("Ошибка при инициализации базы данных: ", err)
	}

	// Проверяем, реально ли база отвечает нам
	err = db.Ping()
	if err != nil {
		log.Fatal("База данных недоступна (Ping failed): ", err)
	}

	log.Println("Успешное подключение к PostgreSQL!")
	return db
}
