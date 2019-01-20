package main

import (
	"github.com/satori/go.uuid"
	"net/http"
)

const (
	ADMIN = iota
	USER
)

type SessionDB struct {
	Sessions map[string]int
}

func NewSessionDB() *SessionDB {
	return &SessionDB{Sessions: make(map[string]int)}
}

func (s *SessionDB) isAdmin(r *http.Request) bool {
	c, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		return false
	} else {
		if v, ok := s.Sessions[c.Value]; ok && v == ADMIN {
			return true
		}
	}
	return false
}

func (s *SessionDB) createSession(user int) *http.Cookie {
	sID, _ := uuid.NewV4()

	s.Sessions[sID.String()] = user

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
