# Практическое занятие №7

## Написание Dockerfile и сборка контейнера

**Студент:** Холодков А.Д.  
**Группа:** ЭФМО-01-25

## Dockerfile

Оба сервиса используют **multi-stage build**. Это даёт компактный итоговый образ (~15 МБ вместо ~500 МБ с Go-тулчейном).

### Stage 1 — builder

| Шаг | Зачем |
|-----|-------|
| `FROM golang:1.22-alpine` | Образ с Go-компилятором |
| `COPY go.mod go.sum ./` | Копируем только модули — Docker кеширует этот слой отдельно |
| `RUN go mod download` | Скачиваем зависимости (кешируется, пока go.mod не изменился) |
| `COPY . .` | Копируем весь исходный код |
| `RUN CGO_ENABLED=0 go build -o /auth ...` | Статическая сборка — бинарник не требует C-библиотек |

### Stage 2 — runner

| Шаг | Зачем |
|-----|-------|
| `FROM alpine:3.19` | Минимальный образ без компилятора |
| `COPY --from=builder /auth .` | Берём только бинарник из предыдущей стадии |
| `EXPOSE 8081` | Документируем порт (не открывает порт, только информирует) |
| `CMD ["./auth"]` | Команда запуска |

## Переменные окружения
Указаны в файле `.env`

### auth

| Переменная | По умолчанию | Описание |
|------------|-------------|----------|
| `AUTH_PORT` | `8081` | HTTP порт сервиса |
| `AUTH_GRPC_PORT` | `50051` | gRPC порт сервиса |

### tasks

| Переменная | По умолчанию | Описание |
|------------|-------------|----------|
| `TASKS_PORT` | `8082` | HTTP порт сервиса |
| `AUTH_GRPC_ADDR` | `localhost:50051` | Адрес auth gRPC. Внутри docker-сети: `auth:50051` |
| `DATABASE_URL` | `postgres://...@localhost:5432/tasks` | DSN подключения к PostgreSQL |

## Взаимодействие сервисов внутри docker-сети

Внутри docker-сети контейнеры обращаются друг к другу **по имени сервиса**:
- tasks → auth: `auth:50051`
- tasks → db: `db:5432`
- nginx → tasks: `tasks:8082`

Использовать `localhost` для обращения к другому контейнеру **нельзя** — у каждого контейнера свой сетевой стек.

## Сборка и запуск

### Полная связка через docker-compose

```powershell
cd deploy
docker compose up -d --build
```

Проверить статус контейнеров:
```powershell
docker compose ps
```

Посмотреть логи:
```powershell
docker logs -f tasks
docker logs -f auth
```

_Скриншот — `docker compose ps`, все контейнеры Up:_

### Сборка образов вручную (из корня репозитория)

```powershell
# Auth
docker build -t techip-auth:0.1 -f services/auth/Dockerfile .

# Tasks
docker build -t techip-tasks:0.1 -f services/tasks/Dockerfile .
```

## Проверка работоспособности

### Получить токен

```powershell
$login = curl.exe -k -s -X POST https://localhost:8443/v1/auth/login `
  -H "Content-Type: application/json" `
  -c cookies.txt `
  -d '{"username":"student","password":"student"}'

$CSRF = ($login | ConvertFrom-Json).csrf_token
Write-Host "CSRF: $CSRF"
```

_Скриншот — успешный ответ логина:_

### Создать задачу

```powershell
curl.exe -k -i -X POST https://localhost:8443/v1/tasks `
  -H "Content-Type: application/json" `
  -H "X-CSRF-Token: $CSRF" `
  -H "X-Request-ID: pz7-001" `
  -b cookies.txt `
  -d '{"title":"Docker task","description":"containerized"}'
```

_Скриншот — ответ 201 Created:_

### Получить список задач

```powershell
curl.exe -k -i https://localhost:8443/v1/tasks `
  -H "X-Request-ID: pz7-002" `
  -b cookies.txt
```

_Скриншот — список задач с поиском по названию:_