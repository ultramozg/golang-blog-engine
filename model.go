package main

import (
	"database/sql"
)

//Post is struct which holds model representation of one post
type Post struct {
	ID    int
	Title string
	Body  string
	Date  string
}

func (p *Post) getPost(db *sql.DB) error {
	return db.QueryRow(`select * from posts where id = ?`, p.ID).Scan(&p.ID, &p.Title, &p.Body, &p.Date)
}

func (p *Post) updatePost(db *sql.DB) error {
	_, err := db.Exec(`update posts set title = $1, body = $2, datepost = $3 where id = $4`, p.Title, p.Body, p.Date, p.ID)
	return err
}

func (p *Post) deletePost(db *sql.DB) error {
	_, err := db.Exec(`delete from posts where id = ?`, p.ID)
	return err
}

func (p *Post) createPost(db *sql.DB) error {
	_, err := db.Exec(`insert into posts (title, body, datepost) values ($1, $2, $3)`, p.Title, p.Body, p.Date)
	return err
}

func getPosts(db *sql.DB, count, start int) ([]Post, error) {
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

//Comment is struct which holds model representation of one comment
type Comment struct {
	PostID    int
	CommentID int
	Name      string
	Date      string
	Data      string
}

func getComments(db *sql.DB, id int) ([]Comment, error) {
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

func (c *Comment) deleteComment(db *sql.DB) error {
	_, err := db.Exec(`delete from comments where commentid = ?`, c.CommentID)
	return err
}

func (c *Comment) createComment(db *sql.DB) error {
	_, err := db.Exec(`insert into comments (postid, name, date, comment) values ($1, $2, $3, $4)`, c.PostID, c.Name, c.Date, c.Data)
	return err
}

func migrateDatabase(db *sql.DB) {
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
	name string not null,
	pass string not null);
	`

	_, err := db.Exec(sql)

	if err != nil {
		panic(err)
	}
}
