package main

import (
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"pz1/shared/logger"
	"pz1/shared/rabbit"
)

func main() {
	log := logger.Must("worker")
	defer log.Sync()

	rabbitURL := os.Getenv("RABBIT_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}
	queueName := os.Getenv("QUEUE_NAME")
	if queueName == "" {
		queueName = rabbit.QueueName
	}

	// ── Подключение к RabbitMQ ───────────────────────────────────────────────
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatal("worker: failed to connect to rabbitmq",
			zap.String("url", rabbitURL),
			zap.Error(err),
		)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("worker: failed to open channel", zap.Error(err))
	}
	defer ch.Close()

	// Объявляем очередь теми же параметрами что и producer.
	// Если очередь уже существует — просто подключаемся к ней.
	_, err = ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		log.Fatal("worker: failed to declare queue",
			zap.String("queue", queueName),
			zap.Error(err),
		)
	}

	// Worker берёт не более 1 сообщения за раз.
	// Следующее сообщение придёт только после ack предыдущего.
	if err := ch.Qos(1, 0, false); err != nil {
		log.Fatal("worker: failed to set qos", zap.Error(err))
	}

	msgs, err := ch.Consume(
		queueName,
		"",    // consumer tag — пустой, RabbitMQ сгенерирует
		false, // auto-ack = false, подтверждаем вручную
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		log.Fatal("worker: failed to register consumer",
			zap.String("queue", queueName),
			zap.Error(err),
		)
	}

	log.Info("worker started, waiting for messages",
		zap.String("queue", queueName),
	)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for msg := range msgs {
			processMessage(msg, log)
		}
	}()

	<-stop
	log.Info("worker shutting down")
}

// processMessage обрабатывает одно сообщение из очереди.
// После успешной обработки отправляет ack.
// При ошибке десериализации — nack без повторной постановки (битое сообщение).
func processMessage(msg amqp.Delivery, log *zap.Logger) {
	var event rabbit.TaskEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Error("worker: failed to unmarshal message",
			zap.String("component", "consumer"),
			zap.String("error", err.Error()),
			zap.ByteString("body", msg.Body),
		)
		// nack с requeue=false — битое сообщение не возвращаем в очередь.
		msg.Nack(false, false)
		return
	}

	log.Info("worker: received event",
		zap.String("component", "consumer"),
		zap.String("event", event.Event),
		zap.String("task_id", event.TaskID),
		zap.String("request_id", event.RequestID),
		zap.String("producer", event.Producer),
		zap.String("ts", event.Timestamp),
	)

	if err := msg.Ack(false); err != nil {
		log.Error("worker: failed to ack message",
			zap.String("component", "consumer"),
			zap.String("error", err.Error()),
		)
	}
}
