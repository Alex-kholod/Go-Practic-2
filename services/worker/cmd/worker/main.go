package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"pz1/shared/logger"
	"pz1/shared/rabbit"
)

func main() {
	log := logger.Must("worker")
	defer log.Sync() //nolint:errcheck

	rabbitURL := os.Getenv("RABBIT_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	var conn *amqp.Connection
	var err error
	for i := 1; i <= 10; i++ {
		conn, err = amqp.Dial(rabbitURL)
		if err == nil {
			break
		}
		log.Warn("rabbitmq not ready, retrying...",
			zap.Int("attempt", i),
			zap.String("error", err.Error()),
		)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatal("failed to connect to rabbitmq after retries", zap.Error(err))
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("failed to open channel", zap.Error(err))
	}
	defer ch.Close()

	// Объявляем очереди — те же параметры что и у producer.
	if err := rabbit.DeclareJobQueues(ch); err != nil {
		log.Fatal("failed to declare queues", zap.Error(err))
	}

	// Prefetch = 1 — берём одно сообщение за раз.
	if err := ch.Qos(1, 0, false); err != nil {
		log.Fatal("failed to set qos", zap.Error(err))
	}

	msgs, err := ch.Consume(
		rabbit.JobQueue,
		"",    // consumer tag
		false, // auto-ack = false
		false, false, false, nil,
	)
	if err != nil {
		log.Fatal("failed to register consumer", zap.Error(err))
	}

	log.Info("worker started, waiting for jobs",
		zap.String("queue", rabbit.JobQueue),
		zap.String("dlq", rabbit.JobDLQ),
		zap.Int("max_attempts", rabbit.MaxAttempts),
	)

	processed := &processedStore{}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for msg := range msgs {
			processJob(msg, ch, processed, log)
		}
	}()

	<-stop
	log.Info("worker shutting down")
}

func processJob(msg amqp.Delivery, ch *amqp.Channel, processed *processedStore, log *zap.Logger) {
	var job rabbit.JobMessage
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		log.Error("worker: failed to unmarshal job",
			zap.String("component", "consumer"),
			zap.String("error", err.Error()),
		)
		// Битое сообщение → DLQ через nack(requeue=false).
		msg.Nack(false, false)
		return
	}

	log.Info("worker: processing job",
		zap.String("component", "consumer"),
		zap.String("job", job.Job),
		zap.String("task_id", job.TaskID),
		zap.String("message_id", job.MessageID),
		zap.Int("attempt", job.Attempt),
	)

	// Проверка идемпотентности
	if processed.has(job.MessageID) {
		log.Info("worker: duplicate message, skipping",
			zap.String("component", "consumer"),
			zap.String("message_id", job.MessageID),
		)
		msg.Ack(false)
		return
	}

	// Имитация "тяжёлой" работы
	workErr := doWork(job, log)

	if workErr == nil {
		// Успех — запоминаем message_id и подтверждаем.
		processed.add(job.MessageID)
		log.Info("worker: job completed",
			zap.String("component", "consumer"),
			zap.String("task_id", job.TaskID),
			zap.String("message_id", job.MessageID),
			zap.Int("attempt", job.Attempt),
		)
		msg.Ack(false)
		return
	}

	// Обработка ошибки: retry или DLQ
	log.Warn("worker: job failed",
		zap.String("component", "consumer"),
		zap.String("task_id", job.TaskID),
		zap.Int("attempt", job.Attempt),
		zap.String("error", workErr.Error()),
	)

	if job.Attempt < rabbit.MaxAttempts {
		// Повторная попытка: публикуем с увеличенным счётчиком.
		job.Attempt++
		body, _ := json.Marshal(job)
		pubErr := ch.Publish(
			"",              // default exchange
			rabbit.JobQueue, // routing key
			false, false,
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				MessageId:    job.MessageID,
				Body:         body,
			},
		)
		if pubErr != nil {
			log.Error("worker: failed to republish for retry",
				zap.String("error", pubErr.Error()),
			)
		} else {
			log.Info("worker: job requeued for retry",
				zap.String("component", "consumer"),
				zap.String("task_id", job.TaskID),
				zap.String("message_id", job.MessageID),
				zap.Int("next_attempt", job.Attempt),
			)
		}
		// Ack исходное — иначе будет дублирование.
		msg.Ack(false)
	} else {
		// Исчерпаны все попытки → DLQ через nack(requeue=false).
		log.Error("worker: max attempts reached, sending to DLQ",
			zap.String("component", "consumer"),
			zap.String("task_id", job.TaskID),
			zap.String("message_id", job.MessageID),
			zap.Int("attempts", job.Attempt),
		)
		msg.Nack(false, false)
	}
}

func doWork(job rabbit.JobMessage, log *zap.Logger) error {
	// Имитируем задержку обработки.
	delay := time.Duration(1+rand.Intn(2)) * time.Second
	time.Sleep(delay)

	// Детерминированная ошибка: task_id оканчивается на "fail".
	if len(job.TaskID) >= 4 && job.TaskID[len(job.TaskID)-4:] == "fail" {
		return fmt.Errorf("task %s marked as always-failing", job.TaskID)
	}

	// Случайная ошибка для демонстрации retry (30%).
	if rand.Float32() < 0.3 {
		return fmt.Errorf("transient error processing task %s", job.TaskID)
	}

	return nil
}

type processedStore struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func (s *processedStore) has(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seen == nil {
		return false
	}
	_, ok := s.seen[id]
	return ok
}

func (s *processedStore) add(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seen == nil {
		s.seen = make(map[string]struct{})
	}
	s.seen[id] = struct{}{}
}
