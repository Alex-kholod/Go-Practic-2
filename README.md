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
<img width="974" height="507" alt="image" src="https://github.com/user-attachments/assets/a93468a3-c837-4f1a-b2c6-2477deffbf25" />


## Создать задачу
```bash
curl -X POST http://localhost:8082/v1/tasks \
  -H "Authorization: Bearer demo-token" \
  -H "Content-Type: application/json" \
  -d '{"title":"Test","description":"microservices"}'
```
<img width="974" height="527" alt="image" src="https://github.com/user-attachments/assets/6d9035b7-9d07-42fc-8ad5-abbaf49ddcb7" />


## Получить список задач
```bash
curl -X GET http://localhost:8082/v1/tasks \
  -H "Authorization: Bearer demo-token"
```
<img width="974" height="582" alt="image" src="https://github.com/user-attachments/assets/4460bea5-2a3c-42b6-bd91-580d2b5accb2" />


# Логи работы
Logs task service
<img width="974" height="224" alt="image" src="https://github.com/user-attachments/assets/d7c7e126-20a6-454c-865c-4cdcdbe1f0d9" />

Log auth service
<img width="974" height="211" alt="image" src="https://github.com/user-attachments/assets/3c7431f2-24b4-40c4-b784-62136cc606eb" />
