package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"pz1/shared/repository"
)

const (
	// keyTask — шаблон ключа для одной задачи.
	keyTask = "tasks:task:%s"
	// keyList — ключ для списка всех задач.
	keyList = "tasks:list"
)

type Cache struct {
	client    *redis.ClusterClient
	log       *zap.Logger
	baseTTL   time.Duration
	jitterMax time.Duration
}

// New создаёт клиент Redis Cluster и проверяет соединение.
func New(addrs []string, password string, baseTTL, jitterMax time.Duration, log *zap.Logger) (*Cache, error) {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    addrs,
		Password: password,

		DialTimeout:  2 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis cluster ping: %w", err)
	}

	return &Cache{
		client:    client,
		log:       log,
		baseTTL:   baseTTL,
		jitterMax: jitterMax,
	}, nil
}

func (c *Cache) ttl() time.Duration {
	jitter := time.Duration(rand.Int63n(int64(c.jitterMax)))
	return c.baseTTL + jitter
}

func (c *Cache) GetTask(ctx context.Context, id string) (repository.Task, bool) {
	key := fmt.Sprintf(keyTask, id)

	val, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		// Cache miss — нормальная ситуация.
		c.log.Debug("cache miss",
			zap.String("component", "cache"),
			zap.String("key", key),
		)
		return repository.Task{}, false
	}
	if err != nil {
		// Redis недоступен — логируем и деградируем в БД.
		c.log.Warn("cache get error, falling back to db",
			zap.String("component", "cache"),
			zap.String("key", key),
			zap.String("error", err.Error()),
		)
		return repository.Task{}, false
	}

	var task repository.Task
	if err := json.Unmarshal(val, &task); err != nil {
		// Битые данные в кэше — логируем и идём в БД.
		c.log.Warn("cache unmarshal error, falling back to db",
			zap.String("component", "cache"),
			zap.String("key", key),
			zap.String("error", err.Error()),
		)
		return repository.Task{}, false
	}

	c.log.Debug("cache hit",
		zap.String("component", "cache"),
		zap.String("key", key),
	)
	return task, true
}

func (c *Cache) SetTask(ctx context.Context, task repository.Task) {
	key := fmt.Sprintf(keyTask, task.ID)

	data, err := json.Marshal(task)
	if err != nil {
		c.log.Error("cache marshal error",
			zap.String("component", "cache"),
			zap.String("key", key),
			zap.String("error", err.Error()),
		)
		return
	}

	ttl := c.ttl()
	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		c.log.Warn("cache set error",
			zap.String("component", "cache"),
			zap.String("key", key),
			zap.String("error", err.Error()),
		)
		return
	}

	c.log.Debug("cache set",
		zap.String("component", "cache"),
		zap.String("key", key),
		zap.Duration("ttl", ttl),
	)
}

func (c *Cache) DeleteTask(ctx context.Context, id string) {
	key := fmt.Sprintf(keyTask, id)
	if err := c.client.Del(ctx, key).Err(); err != nil {
		c.log.Warn("cache delete error",
			zap.String("component", "cache"),
			zap.String("key", key),
			zap.String("error", err.Error()),
		)
		return
	}
	c.log.Debug("cache deleted",
		zap.String("component", "cache"),
		zap.String("key", key),
	)
}

// Вызывается при любой операции создания/изменения/удаления.
func (c *Cache) InvalidateList(ctx context.Context) {
	if err := c.client.Del(ctx, keyList).Err(); err != nil {
		c.log.Warn("cache invalidate list error",
			zap.String("component", "cache"),
			zap.String("key", keyList),
			zap.String("error", err.Error()),
		)
	}
}
