# Этап сборки: используем Golang-образ на базе Alpine для компактной сборки
FROM golang:1.23-alpine AS builder
WORKDIR /app
# Копируем файлы модуля и скачиваем зависимости
COPY go.mod go.sum ./
RUN go mod download
# Копируем весь исходный код
COPY . .
# Собираем бинарник мигратора из каталога cmd/migrator
RUN CGO_ENABLED=0 go build -o migrator ./cmd/migrator

# Финальный образ: используем Ubuntu для более стабильного окружения
FROM ubuntu:22.04
# Обновляем пакеты и устанавливаем сертификаты (необходимы для HTTPS)
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
# Копируем собранный бинарник из предыдущего этапа
COPY --from=builder /app/migrator .
# Копируем конфигурационные файлы и миграции
COPY config config
COPY migrations migrations
# Определяем точку входа: запуск мигратора
ENTRYPOINT ["/app/migrator"]