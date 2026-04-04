# Практическое занятие №15

## Деплой приложения на VPS. Настройка systemd

**Студент:** Холодков А.Д.  
**Группа:** ЭФМО-01-25

## 1. Варианты деплоя

В рамках работы рассмотрены **два подхода**:

### Вариант A — бинарник + systemd
- Go-бинарник собирается локально для `linux/amd64`
- Копируется на VPS через `scp`
- systemd управляет запуском, перезапуском и логами

### Вариант B — Docker + CI/CD (современный подход)
- Код хранится в Git-репозитории
- При пуше автоматически запускается CI/CD
- Собираются Docker-образы
- Образы публикуются в registry
- Сервер автоматически обновляется через `docker-compose`

## 2. Структура директорий на VPS (systemd)

```

/opt/tasks/
├── tasks          ← бинарник (текущая версия)
└── tasks.old      ← предыдущая версия (для отката)

/etc/tasks/
└── tasks.env      ← переменные окружения (права 600)

/etc/systemd/system/
└── tasks.service  ← unit-файл сервиса

```

## 3. Первоначальная настройка VPS

```bash
ssh user@<VPS_IP>

sudo apt update && sudo apt upgrade -y

sudo useradd --system --no-create-home --shell /usr/sbin/nologin tasksuser

sudo mkdir -p /opt/tasks
sudo chown tasksuser:tasksuser /opt/tasks

sudo mkdir -p /etc/tasks
```

## 4. Конфигурация `/etc/tasks/tasks.env`

```env
TASKS_PORT=8082
INSTANCE_ID=tasks-vps
AUTH_GRPC_ADDR=127.0.0.1:50051
DATABASE_URL=postgres://tasks_user:PASSWORD@127.0.0.1:5432/tasks?sslmode=disable
REDIS_ADDRS=127.0.0.1:6379
RABBIT_URL=amqp://tasks_user:PASSWORD@127.0.0.1:5672/
```

```bash
sudo chown root:root /etc/tasks/tasks.env
sudo chmod 600 /etc/tasks/tasks.env
```

## 5. systemd unit `/etc/systemd/system/tasks.service`

```ini
[Unit]
Description=Tasks Service
After=network.target

[Service]
Type=simple
User=tasksuser
WorkingDirectory=/opt/tasks
EnvironmentFile=/etc/tasks/tasks.env
ExecStart=/opt/tasks/tasks
Restart=always
RestartSec=2
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/opt/tasks
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

## 6. Деплой через systemd (Вариант A)

### Сборка

```powershell
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go build -o bin/tasks ./services/tasks/cmd/tasks
```

### Копирование

```bash
scp bin/tasks user@<VPS_IP>:/tmp/tasks
```

### Установка

```bash
sudo mv /tmp/tasks /opt/tasks/tasks
sudo chown tasksuser:tasksuser /opt/tasks/tasks
sudo chmod 755 /opt/tasks/tasks
```


## 7. Управление сервисом

```bash
sudo systemctl daemon-reload
sudo systemctl start tasks
sudo systemctl enable tasks
sudo systemctl status tasks
sudo systemctl restart tasks
sudo systemctl stop tasks
```

## 8. Логи

```bash
sudo journalctl -u tasks -n 30 --no-pager
sudo journalctl -u tasks -f
sudo journalctl -u tasks --since today
```

## 9. Проверка

```bash
curl -i http://127.0.0.1:8082/health
```

## 10. Обновление и откат (systemd)

### Обновление

```bash
sudo systemctl stop tasks
sudo cp /opt/tasks/tasks /opt/tasks/tasks.old
sudo mv /tmp/tasks /opt/tasks/tasks
sudo systemctl start tasks
```

### Откат

```bash
sudo systemctl stop tasks
sudo mv /opt/tasks/tasks.old /opt/tasks/tasks
sudo systemctl start tasks
```


# 11. CI/CD деплой (Вариант B — Docker)

## Общая схема

1. Push в репозиторий
2. CI запускается автоматически
3. Сборка Docker-образов
4. Публикация в registry
5. Подключение к серверу по SSH
6. Обновление контейнеров

## Пример CI/CD пайплайна

```yaml
name: CI/CD Pipeline

on:
  push:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run tests
        run: go test ./...

  docker-build:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
      - name: Build Docker image
        run: docker build -t app:latest .

  deploy:
    runs-on: ubuntu-latest
    needs: docker-build
    steps:
      - name: Deploy via SSH
        run: |
          ssh user@server "
            docker-compose pull
            docker-compose up -d
          "
```

## Что происходит при деплое

* Автоматическая сборка образов
* Публикация в registry
* Подключение к серверу
* Обновление контейнеров
* Перезапуск сервисов без ручного вмешательства


## Преимущества CI/CD

### 1. Автоматизация

Не требуется ручной деплой

### 2. Воспроизводимость

Одинаковое окружение через Docker

### 3. Масштабирование

```bash
docker pull image
```

### 4. Откат

Можно запустить предыдущую версию образа


## Деплой на сервере

```bash
cd /opt/app
docker-compose pull
docker-compose up -d
```


## Логи Docker

```bash
docker logs service -f
docker-compose logs
```


# 12. Сравнение подходов

| Критерий        | systemd | Docker + CI/CD |
| --------------- | ------- | -------------- |
| Автоматизация   | ❌       | ✅              |
| Простота        | ✅       | ⚠️             |
| Масштабирование | ❌       | ✅              |
| Откат           | ⚠️      | ✅              |
| Изоляция        | ❌       | ✅              |
| Современность   | ❌       | ✅              |


# 13. Вывод

В работе рассмотрены два подхода к деплою:

### systemd

Подходит для:

* учебных задач
* простых сервисов
* минимальной инфраструктуры

### CI/CD + Docker

Используется в реальной разработке, так как:

* автоматизирует процессы
* снижает количество ошибок
* обеспечивает стабильность и масштабируемость

**Итог:**
systemd — базовый вариант
CI/CD — современный стандарт индустрии
