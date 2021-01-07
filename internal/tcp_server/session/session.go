package session

import (
	"errors"
	"example.com/kendrick/api"
	"example.com/kendrick/internal/tcp_server/cache"
	"github.com/satori/uuid"
	"time"
)

/**
Session manager handles get/create/delete sessions and handles session timeout
*/

var (
	ERR_SESSION_TIMEOUT = errors.New("Session has timed out")
	ERR_NO_SUCH_SESSION = errors.New("Session doesn't exist")
)

type SessionManager interface {
	GetSession(sid string) (api.Session, error)
	CreateSession(user *api.User) (api.Session, error)
	EditSession(sid string, user *api.User) error
	DeleteSession(sid string) error
	Stop()
}

type SessionMgrStruct struct {
	sessionCache      cache.DBCache
	sessionTimeoutHrs int
}

func NewManager(sessionTimeoutHrs int) (SessionManager, error) {
	sessionCache := cache.NewRedisCache(
		"localhost:6379",
		0,
		time.Duration(sessionTimeoutHrs)*time.Hour,
	)

	return &SessionMgrStruct{
		sessionCache:      sessionCache,
		sessionTimeoutHrs: sessionTimeoutHrs,
	}, nil
}

func (manager *SessionMgrStruct) GetSession(sid string) (api.Session, error) {
	session, err := manager.sessionCache.GetSession(sid)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (manager *SessionMgrStruct) CreateSession(user *api.User) (api.Session, error) {
	session := api.SessionStruct{
		SessID: uuid.NewV4().String(),
		User:   user,
	}
	err := manager.sessionCache.SetSession(session.SessID, &session)
	return &session, err
}

func (manager *SessionMgrStruct) EditSession(sid string, user *api.User) error {
	newSess := api.SessionStruct{
		SessID: sid,
		User:   user,
	}
	err := manager.sessionCache.SetSession(sid, &newSess)
	if err != nil {
		return err
	}
	return nil
}

func (manager *SessionMgrStruct) DeleteSession(sid string) error {
	err := manager.sessionCache.DeleteSession(sid)
	return err
}

func (manager *SessionMgrStruct) Stop() {
	// Do nothing for now
}
