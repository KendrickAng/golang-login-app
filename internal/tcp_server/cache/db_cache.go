package cache

import (
	"example.com/kendrick/api"
)

type DBCache interface {
	GetSession(key string) (api.Session, error) // uuid to username
	SetSession(key string, s api.Session) error
	DeleteSession(key string) error
	GetUser(key string) ([]api.User, error) // username to user info
	SetUser(key string, user []api.User) error
}
