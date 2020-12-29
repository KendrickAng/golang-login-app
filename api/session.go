package api

type Session interface {
	GetSessID() string
	GetUsername() string
	GetNickname() string
	GetPwHash() string
	GetProfilePic() string
}

type SessionStruct struct {
	SessID string
	User   *User
}

func (s *SessionStruct) GetSessID() string {
	return s.SessID
}

func (s *SessionStruct) GetUsername() string {
	return s.User.Username
}

func (s *SessionStruct) GetNickname() string {
	return s.User.Nickname
}

func (s *SessionStruct) GetPwHash() string {
	return s.User.PwHash
}

func (s *SessionStruct) GetProfilePic() string {
	return s.User.ProfilePic
}
