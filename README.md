# ПЗ №6 — GORM, миграции и связи

Сортов Семён Александрович, ЭФМО-01-26. Дата: 25.10.2026.

Минимальный REST-сервис с моделями User, Note и Tag. AutoMigrate создаёт users, notes, tags и note_tags. Связи: User → Notes (1:N), Note ↔ Tag (M:N). Preload загружает автора и теги заметки.

Требования: Go 1.25+, Docker Desktop, PowerShell или curl. Проверено с Go 1.27.2 на Windows 11 amd64 и PostgreSQL 15.

## Запуск

```powershell
docker compose up -d
docker compose exec postgres pg_isready -U postgres -d pz6_gorm
$env:DB_DSN='host=127.0.0.1 user=postgres password=postgres dbname=pz6_gorm port=5436 sslmode=disable'
go run ./cmd/server
```

API слушает порт 8080. Другие практики с этим портом запускаются отдельно. Пароль postgres относится к локальной учебной базе. .env.example содержит пример; переменная задаётся в текущем окне, автоматической загрузки .env нет.

## Запросы в другом окне PowerShell

```powershell
Invoke-RestMethod http://localhost:8080/health
$user = Invoke-RestMethod http://localhost:8080/users -Method Post -ContentType 'application/json' -Body '{"name":"Alice","email":"alice@example.com"}'
$body = @{title='Первая заметка'; content='Текст...'; userId=$user.ID; tags=@('go','gorm')} | ConvertTo-Json
$note = Invoke-RestMethod http://localhost:8080/notes -Method Post -ContentType 'application/json; charset=utf-8' -Body ([Text.Encoding]::UTF8.GetBytes($body))
Invoke-RestMethod ("http://localhost:8080/notes/" + $note.ID) | ConvertTo-Json -Depth 8
docker compose exec postgres psql -U postgres -d pz6_gorm -c '\d notes' -c '\d note_tags'
```

POST /users и POST /notes возвращают 201, GET /notes/{id} — 200 с User и Tags. Повтор email даёт 409; отсутствующие поля и неверный ID — 400; неизвестная заметка — 404. Структуры в ответе используют имена полей ID, Title, User, Tags, как в базовом листинге.

## Проверки и типовые проблемы

```powershell
go test ./...
go vet ./...
go build -o bin/app.exe ./cmd/server
```

- DB_DSN is empty: задать переменную перед запуском.
- Connection refused: дождаться PostgreSQL на порту 5436.
- Email уже существует: использовать другой email либо прочитать ранее созданные данные; ограничение уникальности работает штатно.
- При повторном запуске AutoMigrate сохраняет существующие строки.

В исходном задании имя каталога HTTP не совпадало с импортом в main. Использован существующий каталог internal/http с псевдонимом httpapi. Создание заметки и тегов выполняется в транзакции; неизвестный автор не оставляет лишних тегов.
