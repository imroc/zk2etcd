package record

import (
	"context"
	"github.com/go-redis/redis/v8"
)

var defaultRecord *Record

type Record struct {
	client *redis.Client
	ctx    context.Context
}

func New(redisServer, redisUsername, redisPassword string) *Record {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisServer,
		Password: redisPassword, // no password set
		Username: redisUsername,
		DB:       0, // use default DB
	})
	return &Record{
		client: rdb,
		ctx:    context.Background(),
	}
}

func Get(key string) (value string, exist bool, err error) {
	value, err = defaultRecord.client.Get(defaultRecord.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			err = nil
		}
	} else {
		exist = true
	}
	return
}

func Delete(key string) (err error) {
	err = defaultRecord.client.Del(defaultRecord.ctx, key).Err()
	return
}

func Put(key, value string) (err error) {
	err = defaultRecord.client.Set(defaultRecord.ctx, key, value, 0).Err()
	return
}
