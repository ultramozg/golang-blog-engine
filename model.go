package main

import (
	"database/sql"
)

type Post struct {
	Id    int
	Title string
	Body  string
	Date  string
}

func (p *Post) getPost(db *sql.DB) error {
	return db.QueryRow(`select * from posts where id = ?`, p.Id).Scan(&p.Id, &p.Title, &p.Body, &p.Date)
}

func (p *Post) updatePost(db *sql.DB) error {
	_, err := db.Exec(`update posts set title = $1, body = $2, datepost = $3 where id = $4`, p.Title, p.Body, p.Date, p.Id)
	return err
}

func (p *Post) deletePost(db *sql.DB) error {
	_, err := db.Exec(`delete from posts where id = ?`, p.Id)
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
		if err := rows.Scan(&p.Id, &p.Title, &p.Body, &p.Date); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}

type Comment struct {
	PostId int
	Name   string
	Date   string
	Data   string
}

func getComments(db *sql.DB, id int) ([]Comment, error) {
	rows, err := db.Query(`select postid, name, date, data from comments where id = ? order by id desc;`, id)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []Comment{}

	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.PostId, &c.Name, &c.Date, &c.Data); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}

func (c *Comment) deleteComment(db *sql.DB) error {
	_, err := db.Exec(`delete from comments where postid = ?`, c.PostId)
	return err
}

func (c *Comment) createComment(db *sql.DB) error {
	_, err := db.Exec(`insert into comments (postid, name, date, data) values ($1, $2, $3, $4)`, c.PostId, c.Name, c.Date, c.Data)
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
	name string not null,
	date string not null,
	comment  string not null);
	`

	_, err := db.Exec(sql)

	if err != nil {
		panic(err)
	}
}
