package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	rdb *redis.Client
}

func New(addr string) *RedisStore {
	return &RedisStore{
		rdb: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: "",
			DB:       0,
			Protocol: 2,
		}),
	}
}

func (s *RedisStore) GetSuggestions(ctx context.Context, prefix string, limit int) ([]string, error) {
	suggestions, err := s.rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:   prefix,
		Start: 0,
		Stop:  limit - 1,
		Rev:   true,
	}).Result()
	if err != nil {
		return nil, err
	}

	return suggestions, nil
}

func (s *RedisStore) IncrementFrequency(ctx context.Context, query string) error {
	for i := range query {
		if err := s.rdb.ZIncrBy(ctx, query[:i+1], 1, query).Err(); err != nil {
			return err
		}
	}

	return nil
}
