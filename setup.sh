#!/bin/bash
# deploy/vps/setup.sh
# Первоначальная настройка VPS — запускается один раз на сервере.
# ssh user@VPS_IP "bash -s" < deploy/vps/setup.sh

set -euo pipefail

echo "==> Обновление пакетов..."
sudo apt update && sudo apt upgrade -y

echo "==> Создание системного пользователя tasksuser..."
if ! id tasksuser &>/dev/null; then
  sudo useradd --system --no-create-home --shell /usr/sbin/nologin tasksuser
fi

echo "==> Создание директорий..."
sudo mkdir -p /opt/tasks
sudo chown tasksuser:tasksuser /opt/tasks

sudo mkdir -p /etc/tasks
echo "  Создай /etc/tasks/tasks.env по шаблону из репозитория"
echo "  Права: sudo chown root:root /etc/tasks/tasks.env && sudo chmod 600 /etc/tasks/tasks.env"

echo "==> Копирование tasks.service..."
# Предполагается что файл уже скопирован на VPS
if [ -f /tmp/tasks.service ]; then
  sudo cp /tmp/tasks.service /etc/systemd/system/tasks.service
  sudo systemctl daemon-reload
  sudo systemctl enable tasks
  echo "  tasks.service установлен и включён"
else
  echo "  ВНИМАНИЕ: /tmp/tasks.service не найден, скопируй его вручную"
fi

echo ""
echo "==> Следующие шаги:"
echo "  1. Создать /etc/tasks/tasks.env (по шаблону tasks.env)"
echo "  2. Скопировать бинарник: scp bin/tasks user@VPS:/opt/tasks/tasks"
echo "  3. sudo systemctl start tasks"
echo "  4. sudo systemctl status tasks"
