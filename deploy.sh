#!/bin/bash
# deploy/vps/deploy.sh
# Скрипт деплоя бинарника tasks на VPS.
# Запускать с локальной машины:
#   bash deploy/vps/deploy.sh <VPS_IP> [VPS_USER]
#
# Требует: Go установлен локально, SSH-доступ настроен.

set -euo pipefail

VPS_IP="${1:?Укажи IP VPS: deploy.sh <VPS_IP> [user]}"
VPS_USER="${2:-ubuntu}"
REMOTE="${VPS_USER}@${VPS_IP}"
SERVICE_NAME="tasks"
REMOTE_DIR="/opt/tasks"
BINARY_NAME="tasks"

echo "==> Сборка бинарника для Linux amd64..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -o "bin/${BINARY_NAME}" ./services/tasks/cmd/tasks

echo "==> Копирование бинарника на VPS ${REMOTE}..."
scp "bin/${BINARY_NAME}" "${REMOTE}:/tmp/${BINARY_NAME}"

echo "==> Установка на VPS..."
ssh "${REMOTE}" bash << EOF
  set -e

  # Создаём системного пользователя если нет
  if ! id tasksuser &>/dev/null; then
    sudo useradd --system --no-create-home --shell /usr/sbin/nologin tasksuser
    echo "  Пользователь tasksuser создан"
  fi

  # Создаём директорию
  sudo mkdir -p ${REMOTE_DIR}
  sudo chown tasksuser:tasksuser ${REMOTE_DIR}

  # Заменяем бинарник (с сохранением старого для отката)
  if [ -f "${REMOTE_DIR}/${BINARY_NAME}" ]; then
    sudo cp "${REMOTE_DIR}/${BINARY_NAME}" "${REMOTE_DIR}/${BINARY_NAME}.old"
  fi
  sudo mv /tmp/${BINARY_NAME} ${REMOTE_DIR}/${BINARY_NAME}
  sudo chown tasksuser:tasksuser ${REMOTE_DIR}/${BINARY_NAME}
  sudo chmod 755 ${REMOTE_DIR}/${BINARY_NAME}

  echo "  Бинарник установлен: ${REMOTE_DIR}/${BINARY_NAME}"
EOF

echo "==> Перезапуск сервиса..."
ssh "${REMOTE}" "sudo systemctl restart ${SERVICE_NAME} && sudo systemctl status ${SERVICE_NAME} --no-pager"

echo "==> Готово! Проверка health..."
sleep 2
ssh "${REMOTE}" "curl -s http://127.0.0.1:8082/health || echo 'health endpoint не ответил'"

echo "==> Деплой завершён."
