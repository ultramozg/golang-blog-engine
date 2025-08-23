package model

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v2"
)

// TODO need to delete it as in the seesion.go aleady exists this constant
// ADMIN is identificator constant
// GITHUB is user which is loged in via github
const (
	ADMIN = iota + 1
	GITHUB
)

// Post is struct which holds model representation of one post
type Post struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Date      string `json:"date"`
	Slug      string `json:"slug"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (p *Post) GetPost(db *sql.DB) error {
	return db.QueryRow(`select id, title, body, datepost, COALESCE(slug, ''), COALESCE(created_at, ''), COALESCE(updated_at, '') from posts where id = ?`, p.ID).Scan(&p.ID, &p.Title, &p.Body, &p.Date, &p.Slug, &p.CreatedAt, &p.UpdatedAt)
}

func (p *Post) GetPostBySlug(db *sql.DB) error {
	return db.QueryRow(`select id, title, body, datepost, COALESCE(slug, ''), COALESCE(created_at, ''), COALESCE(updated_at, '') from posts where slug = ?`, p.Slug).Scan(&p.ID, &p.Title, &p.Body, &p.Date, &p.Slug, &p.CreatedAt, &p.UpdatedAt)
}

func (p *Post) UpdatePost(db *sql.DB) error {
	_, err := db.Exec(`update posts set title = $1, body = $2, datepost = $3, updated_at = CURRENT_TIMESTAMP where id = $4`, p.Title, p.Body, p.Date, p.ID)
	return err
}

func (p *Post) DeletePost(db *sql.DB) error {
	_, err := db.Exec(`delete from posts where id = ?`, p.ID)
	return err
}

func (p *Post) CreatePost(db *sql.DB) error {
	result, err := db.Exec(`insert into posts (title, body, datepost, created_at, updated_at) values ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, p.Title, p.Body, p.Date)
	if err != nil {
		return err
	}

	// Get the ID of the newly created post
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = int(id)

	return nil
}

func GetPosts(db *sql.DB, count, start int) ([]Post, error) {
	rows, err := db.Query(`select id, title, substr(body,1,950), datepost, COALESCE(slug, ''), COALESCE(created_at, ''), COALESCE(updated_at, '') from posts order by id desc limit ? offset ?;`, count, start)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []Post{}

	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Title, &p.Body, &p.Date, &p.Slug, &p.CreatedAt, &p.UpdatedAt); err != nil {
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

// GetPostBySlug retrieves a post by its slug
func GetPostBySlug(db *sql.DB, slug string) (*Post, error) {
	post := &Post{Slug: slug}
	err := post.GetPostBySlug(db)
	if err != nil {
		return nil, err
	}
	return post, nil
}

// File is struct which holds model representation of one file
type File struct {
	ID            int    `json:"id"`
	UUID          string `json:"uuid"`
	OriginalName  string `json:"original_name"`
	StoredName    string `json:"stored_name"`
	Path          string `json:"path"`
	Size          int64  `json:"size"`
	MimeType      string `json:"mime_type"`
	DownloadCount int    `json:"download_count"`
	CreatedAt     string `json:"created_at"`
}

func (f *File) GetFile(db *sql.DB) error {
	return db.QueryRow(`select id, uuid, original_name, stored_name, path, size, mime_type, download_count, created_at from files where id = ?`, f.ID).Scan(&f.ID, &f.UUID, &f.OriginalName, &f.StoredName, &f.Path, &f.Size, &f.MimeType, &f.DownloadCount, &f.CreatedAt)
}

func (f *File) GetFileByUUID(db *sql.DB) error {
	return db.QueryRow(`select id, uuid, original_name, stored_name, path, size, mime_type, download_count, created_at from files where uuid = ?`, f.UUID).Scan(&f.ID, &f.UUID, &f.OriginalName, &f.StoredName, &f.Path, &f.Size, &f.MimeType, &f.DownloadCount, &f.CreatedAt)
}

func (f *File) CreateFile(db *sql.DB) error {
	result, err := db.Exec(`insert into files (uuid, original_name, stored_name, path, size, mime_type, download_count, created_at) values ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP)`, f.UUID, f.OriginalName, f.StoredName, f.Path, f.Size, f.MimeType, f.DownloadCount)
	if err != nil {
		return err
	}

	// Get the ID of the newly created file
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	f.ID = int(id)

	return nil
}

func (f *File) DeleteFile(db *sql.DB) error {
	_, err := db.Exec(`delete from files where id = ?`, f.ID)
	return err
}

func (f *File) IncrementDownloadCount(db *sql.DB) error {
	_, err := db.Exec(`update files set download_count = download_count + 1 where id = ?`, f.ID)
	return err
}

func GetFiles(db *sql.DB, limit, offset int) ([]File, error) {
	rows, err := db.Query(`select id, uuid, original_name, stored_name, path, size, mime_type, download_count, created_at from files order by created_at desc limit ? offset ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := []File{}
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.UUID, &f.OriginalName, &f.StoredName, &f.Path, &f.Size, &f.MimeType, &f.DownloadCount, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

// Comment is struct which holds model representation of one comment
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
	datepost string not null,
	slug text unique,
	created_at datetime default current_timestamp,
	updated_at datetime default current_timestamp);

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

	create table if not exists files (
	id integer primary key autoincrement,
	uuid text unique not null,
	original_name text not null,
	stored_name text not null,
	path text not null,
	size integer not null,
	mime_type text not null,
	download_count integer default 0,
	created_at datetime default current_timestamp);
	`

	_, err := db.Exec(sql)

	if err != nil {
		panic(err)
	}

	// Run additional migrations for existing databases
	MigrateExistingDatabase(db)
}

// MigrateExistingDatabase adds new columns to existing posts table
func MigrateExistingDatabase(db *sql.DB) {
	// Check if slug column exists
	var columnExists int
	err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('posts') WHERE name='slug'").Scan(&columnExists)
	if err == nil && columnExists == 0 {
		// Add slug column
		_, err = db.Exec("ALTER TABLE posts ADD COLUMN slug TEXT")
		if err != nil {
			log.Println("Warning: Could not add slug column:", err)
		}
	}

	// Check if created_at column exists
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('posts') WHERE name='created_at'").Scan(&columnExists)
	if err == nil && columnExists == 0 {
		// Add created_at column (SQLite doesn't support DEFAULT CURRENT_TIMESTAMP in ALTER TABLE)
		_, err = db.Exec("ALTER TABLE posts ADD COLUMN created_at DATETIME")
		if err != nil {
			log.Println("Warning: Could not add created_at column:", err)
		} else {
			// Set default values for existing rows
			_, err = db.Exec("UPDATE posts SET created_at = CURRENT_TIMESTAMP WHERE created_at IS NULL")
			if err != nil {
				log.Println("Warning: Could not set default created_at values:", err)
			}
		}
	}

	// Check if updated_at column exists
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('posts') WHERE name='updated_at'").Scan(&columnExists)
	if err == nil && columnExists == 0 {
		// Add updated_at column
		_, err = db.Exec("ALTER TABLE posts ADD COLUMN updated_at DATETIME")
		if err != nil {
			log.Println("Warning: Could not add updated_at column:", err)
		} else {
			// Set default values for existing rows
			_, err = db.Exec("UPDATE posts SET updated_at = CURRENT_TIMESTAMP WHERE updated_at IS NULL")
			if err != nil {
				log.Println("Warning: Could not set default updated_at values:", err)
			}
		}
	}

	// Create unique index on slug column for performance and uniqueness
	_, err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_posts_slug ON posts(slug)")
	if err != nil {
		log.Println("Warning: Could not create unique slug index:", err)
	}

	// Generate slugs for existing posts that don't have them
	GenerateSlugsForExistingPosts(db)
}

// GenerateSlugsForExistingPosts creates slugs for posts that don't have them
func GenerateSlugsForExistingPosts(db *sql.DB) {
	// Create a simple slug service for migration
	slugService := &migrationSlugService{db: db}

	// Get all posts without slugs - collect them first to avoid cursor issues
	rows, err := db.Query("SELECT id, title FROM posts WHERE slug IS NULL OR slug = ''")
	if err != nil {
		log.Println("Warning: Could not query posts for slug generation:", err)
		return
	}

	// Collect all posts that need slugs
	type postData struct {
		id    int
		title string
	}
	var posts []postData

	for rows.Next() {
		var p postData
		if err := rows.Scan(&p.id, &p.title); err != nil {
			log.Println("Warning: Could not scan post for slug generation:", err)
			continue
		}
		posts = append(posts, p)
	}
	rows.Close()

	// Now update each post
	for _, post := range posts {
		// Generate and ensure unique slug
		slug := slugService.GenerateSlug(post.title)
		uniqueSlug := slugService.EnsureUniqueSlug(slug, post.id)

		// Update the post with the generated slug
		_, err = db.Exec("UPDATE posts SET slug = ? WHERE id = ?", uniqueSlug, post.id)
		if err != nil {
			log.Printf("Warning: Could not update slug for post %d: %v", post.id, err)
		}
	}
}

// User struct holds information about user
type User struct {
	Type int
	Name string
}

func (u *User) IsUserExist(db *sql.DB) bool {
	status := 0
	if err := db.QueryRow(`select count(*) from users where name = ?`, u.Name).Scan(&status); err != nil {
		log.Println("Database scan error:", err)
		return false
	}
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

// Course holds information about courses which is located under data/courses.yml
type Info struct {
	Title       string `yaml:"title"`
	Link        string `yaml:"link"`
	Description string `yaml:"description,omitempty"`
}

type Infos struct {
	List []Info `yaml:"infos,flow"`
}

func ConverYamlToStruct(path string) (i Infos, err error) {
	// Sanitize path to prevent directory traversal attacks
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return i, errors.New("invalid file path: directory traversal not allowed")
	}

	// Additional validation: only allow .yml and .yaml files
	ext := filepath.Ext(cleanPath)
	if ext != ".yml" && ext != ".yaml" {
		return i, errors.New("invalid file type: only YAML files are allowed")
	}

	// For production use, ensure the path is within expected directories
	// Allow data/ directory and temp files for testing
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return i, errors.New("invalid file path: cannot resolve absolute path")
	}

	// Check if it's a temp file (for testing) or in data directory
	isTempFile := strings.Contains(absPath, os.TempDir())
	isDataFile := strings.Contains(absPath, "data/") || strings.HasSuffix(absPath, ".yml") || strings.HasSuffix(absPath, ".yaml")

	if !isTempFile && !isDataFile {
		return i, errors.New("invalid file path: access denied")
	}

	b, err := os.ReadFile(cleanPath)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(b, &i)
	if err != nil {
		return
	}
	return
}

// migrationSlugService is a simple implementation for migration purposes
type migrationSlugService struct {
	db *sql.DB
}

func (s *migrationSlugService) GenerateSlug(title string) string {
	if title == "" {
		return "untitled"
	}

	// Simple slug generation for migration
	slug := strings.ToLower(title)

	// Handle common accented characters
	replacements := map[string]string{
		"é": "e", "è": "e", "ê": "e", "ë": "e",
		"á": "a", "à": "a", "â": "a", "ä": "a", "ã": "a",
		"í": "i", "ì": "i", "î": "i", "ï": "i",
		"ó": "o", "ò": "o", "ô": "o", "ö": "o", "õ": "o",
		"ú": "u", "ù": "u", "û": "u", "ü": "u",
		"ç": "c", "ñ": "n",
	}

	for accented, replacement := range replacements {
		slug = strings.ReplaceAll(slug, accented, replacement)
	}

	// Remove special characters except spaces
	slug = regexp.MustCompile(`[^a-z0-9\s]`).ReplaceAllString(slug, "")
	// Replace spaces with hyphens
	slug = regexp.MustCompile(`\s+`).ReplaceAllString(slug, "-")
	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	if slug == "" {
		return "untitled"
	}

	if len(slug) > 100 {
		slug = slug[:100]
		slug = strings.TrimRight(slug, "-")
	}

	return slug
}

func (s *migrationSlugService) EnsureUniqueSlug(slug string, postID int) string {
	originalSlug := slug
	counter := 1

	for !s.IsSlugUnique(slug, postID) {
		slug = fmt.Sprintf("%s-%d", originalSlug, counter)
		counter++
		if counter > 1000 {
			break
		}
	}

	return slug
}

func (s *migrationSlugService) IsSlugUnique(slug string, excludePostID int) bool {
	var count int
	var err error

	if excludePostID > 0 {
		err = s.db.QueryRow("SELECT COUNT(*) FROM posts WHERE slug = ? AND id != ?", slug, excludePostID).Scan(&count)
	} else {
		err = s.db.QueryRow("SELECT COUNT(*) FROM posts WHERE slug = ?", slug).Scan(&count)
	}

	if err != nil {
		return false
	}

	return count == 0
}
