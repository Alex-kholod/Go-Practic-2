# Практическое занятие №2

## Тема: gRPC: создание простого микросервиса, вызовы методов

**Студент:** Холодков А.Д.  
**Группа:** ЭФМО-01-25

### Установка и запуск
1. Клонирование репозитория (только ветка текущей практики)
```bash
git clone -b pz2 --single-branch https://github.com/Alex-kholod/Go-Practic-2.git
```
2. Переход в директорию проекта
```bash
cd Go-Practic-2
```

### Принцип работы
Переделана авторизация с HTTP на gRPC. Добавлен файл proto/auth.proto описывающий контакт взаимодействия между клиентом и сервером
Для генерации кода gRPC сервера авторизации была использована команда:
```bash
protoc --go_out=. --go-grpc_out=. proto/auth.proto
```

# Инструкция по запуску
После запуска приложения можно использовать его endpoint's для взаимодействия.
## 1. Запуск Auth-сервиса
```bash
cd services/auth
export AUTH_PORT=8081
go run ./cmd/auth
```

## 2. Запуск Tasks-сервиса
```bash
cd services/tasks
export TASKS_PORT=8082
export AUTH_BASE_URL=http://localhost:8081
go run ./cmd/tasks
```

# Примеры выполнения запросов к API

## Получить токен
```bash
curl -X POST http://localhost:8081/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"student","password":"student"}'
```

## Создать задачу
```bash
curl -X POST http://localhost:8082/v1/tasks \
  -H "Authorization: Bearer demo-token" \
  -H "Content-Type: application/json" \
  -d '{"title":"Test","description":"microservices"}'
```

## Получить список задач
```bash
curl -X GET http://localhost:8082/v1/tasks \
  -H "Authorization: Bearer demo-token"
```

# Логи работы
Logs task service
Log auth service