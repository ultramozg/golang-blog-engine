package session

import (
	"net/http"
	"sync"

	uuid "github.com/satori/go.uuid"
	"github.com/ultramozg/golang-blog-engine/model"
)

// ADMIN is identificator constant
// GITHUB is user which is loged in via github
const (
	ADMIN = iota + 1
	GITHUB
)

// SessionDB is a thread-safe map which holds active sessions
type SessionDB struct {
	sessions map[string]model.User
	mutex    sync.RWMutex
}

// NewSessionDB generate new SessionDB struct
func NewSessionDB() *SessionDB {
	return &SessionDB{
		sessions: make(map[string]model.User),
	}
}

func (s *SessionDB) IsAdmin(r *http.Request) bool {
	c, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		return false
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if v, ok := s.sessions[c.Value]; ok && v.Type == ADMIN {
		return true
	}
	return false
}

func (s *SessionDB) IsLoggedin(r *http.Request) bool {
	c, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		return false
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if _, ok := s.sessions[c.Value]; ok {
		return true
	}
	return false
}

func (s *SessionDB) CreateSession(u model.User) *http.Cookie {
	sID := uuid.NewV4()

	s.mutex.Lock()
	s.sessions[sID.String()] = u
	s.mutex.Unlock()

	c := &http.Cookie{
		Name:  "session",
		Value: sID.String(),
	}
	return c
}

func (s *SessionDB) DelSession(session string) *http.Cookie {
	s.mutex.Lock()
	delete(s.sessions, session)
	s.mutex.Unlock()

	c := &http.Cookie{
		Name:   "session",
		Value:  "",
		MaxAge: -1,
	}
	return c
}

// GetSession retrieves a session by ID (for testing purposes)
func (s *SessionDB) GetSession(sessionID string) (model.User, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	user, exists := s.sessions[sessionID]
	return user, exists
}

// Len returns the number of active sessions (for testing purposes)
func (s *SessionDB) Len() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return len(s.sessions)
}
