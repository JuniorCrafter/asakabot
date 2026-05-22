# ЭТАП 1: Сборка кода (Builder)
FROM golang:alpine AS builder

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем файлы списка библиотек и скачиваем их
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь остальной код
COPY . .

# Компилируем программу в один бинарный файл с названием 'bot'
RUN CGO_ENABLED=0 GOOS=linux go build -o bot cmd/bot/main.go


# ЭТАП 2: Создание финального легкого образа
FROM alpine:latest

WORKDIR /root/

# Копируем только готовый бинарник из первого этапа
COPY --from=builder /app/bot .

# Копируем файл настроек
COPY .env .

# Указываем команду для запуска при старте контейнера
CMD ["./bot"]