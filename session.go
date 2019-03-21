package main

import (
	"github.com/satori/go.uuid"
	"net/http"
)

const (
	ADMIN = iota
	GITHUB
)

type User struct {
	userType int
	userName string
}

type SessionDB struct {
	Sessions map[string]User
}

func NewSessionDB() *SessionDB {
	return &SessionDB{Sessions: make(map[string]User)}
}

func (s *SessionDB) isAdmin(r *http.Request) bool {
	c, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		return false
	} else {
		if v, ok := s.Sessions[c.Value]; ok && v.userType == ADMIN {
			return true
		}
	}
	return false
}

func (s *SessionDB) isLoggedin(r *http.Request) bool {
	c, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		return false
	} else {
		if v, ok := s.Sessions[c.Value]; ok && v.userType == GITHUB {
			return true
		}
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
