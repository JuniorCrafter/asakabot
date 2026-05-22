package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken    string
	DBUrl       string
	RabbitMQUrl string // Новое поле
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Файл .env не найден")
	}

	return &Config{
		BotToken:    os.Getenv("BOT_TOKEN"),
		DBUrl:       os.Getenv("DB_URL"),
		RabbitMQUrl: os.Getenv("RABBITMQ_URL"), // Читаем из .env
	}
}
