package session

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/ultramozg/golang-blog-engine/model"
)

//ADMIN is identificator constant
//GITHUB is user which is loged in via github
const (
	ADMIN = iota + 1
	GITHUB
)

type UserSession struct {
	user  model.User
	token string
}

//SessionDB is just a map which holds active sessions
type SessionDB map[string]*UserSession

//NewSessionDB generate new SessionDB struct
func NewSessionDB() SessionDB {
	return make(map[string]*UserSession)
}

func (s SessionDB) SetToken(r *http.Request, token string) bool {
	c, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		return false
	}
	if v, ok := s[c.Value]; ok {
		v.token = token
		return true
	}
	return false
}

func (s SessionDB) IsAdmin(r *http.Request) bool {
	c, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		return false
	}
	if v, ok := s[c.Value]; ok && v.user.Type == ADMIN {
		return true
	}
	return false
}

func (s SessionDB) IsLoggedin(r *http.Request) bool {
	c, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		return false
	}
	if _, ok := s[c.Value]; ok {
		return true
	}
	return false
}

func (s SessionDB) CreateSession(u model.User) *http.Cookie {
	sID := uuid.NewV4()

	s[sID.String()] = &UserSession{user: u}

	c := &http.Cookie{
		Name:  "session",
		Value: sID.String(),
	}
	return c
}

func (s SessionDB) DelSession(session string) *http.Cookie {
	delete(s, session)

	c := &http.Cookie{
		Name:   "session",
		Value:  "",
		MaxAge: -1,
	}
	return c
}

func GenerateNewToken() string {
	crutime := time.Now().Unix()
	h := md5.New()
	io.WriteString(h, strconv.FormatInt(crutime, 10))
	token := fmt.Sprintf("%x", h.Sum(nil))
	return token
}
