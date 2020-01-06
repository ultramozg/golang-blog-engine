package main

import (
	"net/http"

	uuid "github.com/satori/go.uuid"
)

//ADMIN is identificator constant
//GITHUB is user which is loged in via github
const (
	ADMIN = iota + 1
	GITHUB
)

//User struct holds information about user
type User struct {
	userType int
	userName string
}

//SessionDB is just a map which holds active sessions
type SessionDB struct {
	Sessions map[string]User
}

//NewSessionDB generate new SessionDB struct
func NewSessionDB() *SessionDB {
	return &SessionDB{Sessions: make(map[string]User)}
}

func (s *SessionDB) isAdmin(r *http.Request) bool {
	c, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		return false
	}
	if v, ok := s.Sessions[c.Value]; ok && v.userType == ADMIN {
		return true
	}
	return false
}

func (s *SessionDB) isLoggedin(r *http.Request) bool {
	c, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		return false
	}
	if _, ok := s.Sessions[c.Value]; ok {
		return true
	}
	return false
}

func (s *SessionDB) createSession(u User) *http.Cookie {
	sID, _ := uuid.NewV4()

	s.Sessions[sID.String()] = u

	c := &http.Cookie{
		Name:  "session",
		Value: sID.String(),
	}
	return c
}

func (s *SessionDB) delSession(session string) *http.Cookie {
	delete(s.Sessions, session)

	c := &http.Cookie{
		Name:   "session",
		Value:  "",
		MaxAge: -1,
	}
	return c
}
