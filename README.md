# Практическое занятие №1

## Тема: Разделение монолита на 2 микросервиса. Взаимодействие через HTTP

**Студент:** Холодков А.Д.  
**Группа:** ЭФМО-01-25

### Установка и запуск
1. Клонирование репозитория (только ветка текущей практики)
```bash
git clone -b pz1 --single-branch https://github.com/Alex-kholod/Go-Practic-2.git
```
2. Переход в директорию проекта
```bash
cd Go-Practic-2
```

### Принцип работы
После запуска приложения можно использовать его endpoint's для взаимодействия.
# Инструкция по запуску

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
<img width="974" height="465" alt="image" src="https://github.com/user-attachments/assets/dca5c11d-5cc4-456c-a1a2-9ffd256f559c" />


## Создать задачу
```bash
curl -X POST http://localhost:8082/v1/tasks \
  -H "Authorization: Bearer demo-token" \
  -H "Content-Type: application/json" \
  -d '{"title":"Test","description":"microservices"}'
```
<img width="974" height="492" alt="image" src="https://github.com/user-attachments/assets/77e7c2e0-7351-436e-98cb-d2e6df6db080" />


## Получить список задач
```bash
curl -X GET http://localhost:8082/v1/tasks \
  -H "Authorization: Bearer demo-token"
```
<img width="974" height="648" alt="image" src="https://github.com/user-attachments/assets/b480c6ac-cc03-4d35-97dd-603d4ec056f5" />


# Логи с X-Request-id 
Log auth service
<img width="974" height="269" alt="image" src="https://github.com/user-attachments/assets/014aaf15-104a-4341-9878-26d5bf027d0d" />
Log tasks service
<img width="974" height="268" alt="image" src="https://github.com/user-attachments/assets/d3255b75-8883-4949-a19f-53beb3b8e4d2" />
