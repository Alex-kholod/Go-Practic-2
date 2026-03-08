# Практическое занятие №6

## Защита от CSRF/XSS. Работа с Secure Cookies

**Студент:** Холодков А.Д.  
**Группа:** ЭФМО-01-25

## Cookies: флаги и назначение

После успешного логина сервис `auth` выдаёт две cookie:

| Cookie | HttpOnly | Secure | SameSite | Max-Age | Назначение |
|--------|----------|--------|----------|---------|------------|
| `session` | да | да | Lax | 3600 с | Сессия пользователя. JS не может прочитать — защита от кражи при XSS |
| `csrf_token` | нет | да | Lax | 3600 с | CSRF-токен. JS читает и отправляет в заголовке `X-CSRF-Token` |

**Почему csrf_token не HttpOnly:** фронт должен прочитать значение и передать его в заголовке запроса — это основа Double Submit Cookie паттерна.

## CSRF защита — Double Submit Cookie

### Принцип работы

1. При логине сервер выдаёт `csrf_token` cookie и возвращает то же значение в теле ответа.
2. На каждый `POST` / `PATCH` / `DELETE` клиент обязан передать заголовок `X-CSRF-Token` со значением токена.
3. Middleware сравнивает значение из cookie и из заголовка. Не совпало → **403 Forbidden**.

### Почему это защищает

Вредоносный сайт может заставить браузер отправить cookie жертвы, но **не может прочитать** cookie другого домена — значит, не может сформировать корректный заголовок `X-CSRF-Token`.

### Цепочка middleware в tasks

```
RequestID → Metrics → SecurityHeaders → CSRF → AccessLog → handlers
```

---

## XSS: меры защиты

### Санитизация полей

Поля `title` и `description` при сохранении проходят через `SanitizeText()`:

```go
// Было (опасно):
t.Description = userInput  // "<script>alert(1)</script>"

// Стало (безопасно):
t.Description = middleware.SanitizeText(userInput)
// результат: "&lt;script&gt;alert(1)&lt;/script&gt;"
```

Замены: `<` → `&lt;`, `>` → `&gt;`, `"` → `&quot;`, `'` → `&#39;`.

### Security Headers

На все ответы добавляются заголовки:

| Заголовок | Значение | Цель |
|-----------|----------|------|
| `Content-Security-Policy` | `default-src 'none'` | Запрещает загрузку внешних скриптов/стилей |
| `X-Content-Type-Options` | `nosniff` | Браузер не угадывает MIME-тип |
| `X-Frame-Options` | `DENY` | Запрет встраивания в iframe (Clickjacking) |

## Запуск

```powershell
# Сервис 1 — auth
$env:AUTH_PORT = "8081"; $env:AUTH_GRPC_PORT = "50051"
go run ./services/auth/cmd/auth

# Сервис 2 — tasks
$env:TASKS_PORT = "8082"; $env:AUTH_GRPC_ADDR = "localhost:50051"
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/tasks?sslmode=disable"
go run ./services/tasks/cmd/tasks

# Окружение 4 — NGINX + БД (HTTPS)
cd deploy/tls
docker compose up -d nginx
```

## Проверка

### Логин — получение cookies

```powershell
curl.exe -k -i -X POST https://localhost:8443/v1/auth/login `
  -H "Content-Type: application/json" `
  -c cookies.txt `
  -d '{"username":"student","password":"student"}'
```

В ответе будут установлены две cookie и возвращён `csrf_token` в теле JSON.

_Скриншот — ответ с заголовками Set-Cookie:_

### POST без CSRF токена → ожидается 403

```powershell
curl.exe -k -i -X POST https://localhost:8443/v1/tasks `
  -H "Content-Type: application/json" `
  -b cookies.txt `
  -d '{"title":"CSRF test","description":"no header"}'
```

_Скриншот — ответ 403 Forbidden:_

### POST с CSRF токеном

Прочитать токен из ответа логина и передать в заголовке:

```powershell
# Сохранить csrf_token из ответа логина
$login = curl.exe -k -s -X POST https://localhost:8443/v1/auth/login `
  -H "Content-Type: application/json" `
  -c cookies.txt `
  -d '{"username":"student","password":"student"}'

$CSRF = ($login | ConvertFrom-Json).csrf_token

# Создать задачу с токеном
curl.exe -k -i -X POST https://localhost:8443/v1/tasks `
  -H "Content-Type: application/json" `
  -H "X-CSRF-Token: $CSRF" `
  -b cookies.txt `
  -d '{"title":"CSRF ok","description":"with token"}'
```

_Скриншот — ответ 201 Created:_

### Демонстрация XSS санитизации

```powershell
curl.exe -k -i -X POST https://localhost:8443/v1/tasks `
  -H "Content-Type: application/json" `
  -H "X-CSRF-Token: $CSRF" `
  -b cookies.txt `
  -d '{"title":"XSS test","description":"<script>alert(1)</script>"}'
```

В сохранённой задаче `description` будет содержать `&lt;script&gt;alert(1)&lt;/script&gt;` — тег не выполнится при отображении в браузере.

_Скриншот — сохранённое значение с экранированными тегами:_