package cache

import "example.com/kendrick/api"

type DBCache interface {
	GetSession(key string) (*[]api.Session, error) // uuid to username
	SetSession(key string, value interface{}) error
	GetUser(key string) (*[]api.User, error) // username to user info
	SetUser(key string, user *[]api.User) error
}
