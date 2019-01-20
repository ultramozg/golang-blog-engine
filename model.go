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
