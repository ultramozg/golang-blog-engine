package model

import (
	"database/sql"
	"log"

	"golang.org/x/crypto/bcrypt"
)

//TODO need to delete it as in the seesion.go aleady exists this constant
//ADMIN is identificator constant
//GITHUB is user which is loged in via github
const (
	ADMIN = iota + 1
	GITHUB
)

//Post is struct which holds model representation of one post
type Post struct {
	ID    int
	Title string
	Body  string
	Date  string
}

func (p *Post) GetPost(db *sql.DB) error {
	return db.QueryRow(`select * from posts where id = ?`, p.ID).Scan(&p.ID, &p.Title, &p.Body, &p.Date)
}

func (p *Post) UpdatePost(db *sql.DB) error {
	_, err := db.Exec(`update posts set title = $1, body = $2, datepost = $3 where id = $4`, p.Title, p.Body, p.Date, p.ID)
	return err
}

func (p *Post) DeletePost(db *sql.DB) error {
	_, err := db.Exec(`delete from posts where id = ?`, p.ID)
	return err
}

func (p *Post) CreatePost(db *sql.DB) error {
	_, err := db.Exec(`insert into posts (title, body, datepost) values ($1, $2, $3)`, p.Title, p.Body, p.Date)
	return err
}

func GetPosts(db *sql.DB, count, start int) ([]Post, error) {
	rows, err := db.Query(`select id, title, substr(body,1,950), datepost from posts order by id desc limit ? offset ?;`, count, start)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []Post{}

	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Title, &p.Body, &p.Date); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}

func CountPosts(db *sql.DB) int {
	var c int
	err := db.QueryRow(`select count(*) from posts`).Scan(&c)
	if err != nil {
		log.Println(err)
	}
	return c
}

//Comment is struct which holds model representation of one comment
type Comment struct {
	PostID    int
	CommentID int
	Name      string
	Date      string
	Data      string
}

func GetComments(db *sql.DB, id int) ([]Comment, error) {
	rows, err := db.Query(`select postid, commentid, name, date, comment from comments where postid = ? order by postid desc;`, id)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []Comment{}

	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.PostID, &c.CommentID, &c.Name, &c.Date, &c.Data); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}

func (c *Comment) DeleteComment(db *sql.DB) error {
	_, err := db.Exec(`delete from comments where commentid = ?`, c.CommentID)
	return err
}

func (c *Comment) CreateComment(db *sql.DB) error {
	_, err := db.Exec(`insert into comments (postid, name, date, comment) values ($1, $2, $3, $4)`, c.PostID, c.Name, c.Date, c.Data)
	return err
}

func MigrateDatabase(db *sql.DB) {
	sql := `
	create table if not exists posts (
	id integer primary key autoincrement,
	title string not null,
	body string not null,
	datepost string not null);

	create table if not exists comments (
	postid integer not null,
	commentid integer primary key autoincrement,
	name string not null,
	date string not null,
	comment  string not null);

	create table if not exists users (
	id integer primary key autoincrement,
	name string not null unique,
	type integer not null,
	pass string not null);
	`

	_, err := db.Exec(sql)

	if err != nil {
		panic(err)
	}
}

//User struct holds information about user
type User struct {
	Type int
	Name string
}

func (u *User) IsUserExist(db *sql.DB) bool {
	status := 0
	db.QueryRow(`select count(*) from users where name = ?`, u.Name).Scan(&status)
	if int(status) != 0 {
		return true
	}
	return false
}

func (u *User) CreateUser(db *sql.DB, pswd string) error {
	_, err := db.Exec(`insert into users (name, type, pass) values ($1, $2, $3)`, "admin", u.Type, pswd)
	return err
}

func (u *User) IsAdmin(db *sql.DB) bool {
	var userType int
	err := db.QueryRow(`select type from users where name = ?`, u.Name).Scan(&userType)
	if err != nil {
		log.Println("Error: can't fetch user data :", err)
	}
	if userType == ADMIN {
		return true
	}
	return false
}

func (u *User) CheckCredentials(db *sql.DB, pswd string) bool {
	//Converting the passwords into bytes
	hashedPwd := ""
	err := db.QueryRow(`select pass from users where name = ?`, u.Name).Scan(&hashedPwd)
	if err != nil {
		log.Println("Unable to login, no such user", u.Name)
		return false
	}

	byteHash := []byte(hashedPwd)
	bytePassword := []byte(pswd)

	err = bcrypt.CompareHashAndPassword(byteHash, bytePassword)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}
