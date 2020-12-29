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
	ttl    time.Duration
}

var ctx context.Context = context.TODO()

func NewRedisCache(host string, db int, ttl time.Duration) *redisCache {
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
		LocalCache: rcache.NewTinyLFU(1000, ttl),
	})

	return &redisCache{
		host:   host,
		db:     db,
		client: mycache,
		ttl:    ttl,
	}
}

// Gets the user associated with a session id (the key).
func (cache *redisCache) GetSession(key string) (api.Session, error) {
	var s api.SessionStruct
	err := cache.client.Get(ctx, key, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (cache *redisCache) SetSession(uuid string, s api.Session) error {
	err := cache.client.Set(&rcache.Item{
		Ctx:   ctx,
		Key:   uuid,
		Value: s,
		TTL:   cache.ttl,
	})
	return err
}

func (cache *redisCache) DeleteSession(sid string) error {
	err := cache.client.Delete(ctx, sid)
	return err
}

func (cache *redisCache) GetUser(key string) ([]api.User, error) {
	var users []api.User
	err := cache.client.Get(ctx, key, &users)
	if err != nil {
		return nil, err
	}
	return users, err
}

func (cache *redisCache) SetUser(username string, user []api.User) error {
	err := cache.client.Set(&rcache.Item{
		Ctx:   ctx,
		Key:   username,
		Value: user,
		TTL:   cache.ttl,
	})
	return err
}
