package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Queue interface {
	Enqueue(ctx context.Context, notificationID string) error
	Dequeue(ctx context.Context, timeout time.Duration) (string, error)
}

type redisQueue struct {
	client *redis.Client
	key    string
}

func NewRedisQueue(addr, key string) (Queue, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("connect to redis: %w", err)
	}
	return &redisQueue{client: client, key: key}, nil
}

func (q *redisQueue) Enqueue(ctx context.Context, notificationID string) error {
	return q.client.LPush(ctx, q.key, notificationID).Err()
}

func (q *redisQueue) Dequeue(ctx context.Context, timeout time.Duration) (string, error) {
	result, err := q.client.BRPop(ctx, timeout, q.key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	// BRPop returns [key, value]
	return result[1], nil
}
