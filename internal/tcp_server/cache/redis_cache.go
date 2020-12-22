package cache

import (
	"context"
	"example.com/kendrick/api"
	rcache "github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
	"time"
)

type redisCache struct {
	host   string
	db     int
	client *rcache.Cache
}

const TTL = time.Minute

var ctx context.Context = context.TODO()

func NewRedisCache(host string, db int) *redisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:       host,
		DB:         db,
		Password:   "", // no password set
		MaxRetries: 3,
	})
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Panicln(err)
	}

	mycache := rcache.New(&rcache.Options{
		Redis:      rdb,
		LocalCache: rcache.NewTinyLFU(1000, TTL),
	})

	return &redisCache{
		host:   host,
		db:     db,
		client: mycache,
	}
}

// Gets the username associated with a session id (the key).
func (cache *redisCache) GetSession(key string) (*[]api.Session, error) {
	var sessions []api.Session
	err := cache.client.Get(ctx, key, &sessions)
	if err != nil {
		return nil, err
	}
	return &sessions, nil
}

func (cache *redisCache) SetSession(uuid string, rows interface{}) error {
	err := cache.client.Set(&rcache.Item{
		Ctx:   ctx,
		Key:   uuid,
		Value: rows,
		TTL:   TTL,
	})
	return err
}

func (cache *redisCache) GetUser(key string) (*[]api.User, error) {
	var users []api.User
	err := cache.client.Get(ctx, key, &users)
	if err != nil {
		return nil, err
	}
	return &users, err
}

func (cache *redisCache) SetUser(username string, user *[]api.User) error {
	err := cache.client.Set(&rcache.Item{
		Ctx:   ctx,
		Key:   username,
		Value: user,
		TTL:   TTL,
	})
	return err
}
