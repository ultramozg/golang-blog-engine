package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/ultramozg/golang-blog-engine/model"
	_ "modernc.org/sqlite"
)

func main() {
	// Get database URI from environment or use default
	dbURI := os.Getenv("DBURI")
	if dbURI == "" {
		dbURI = "file:database/database.sqlite"
	}

	// Connect to database
	db, err := sql.Open("sqlite", dbURI)
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}
	defer db.Close()

	// Get all posts that don't have meta descriptions or keywords
	rows, err := db.Query(`
		SELECT id, title, body, datepost, slug, created_at, updated_at, 
		       COALESCE(meta_description, '') as meta_description, 
		       COALESCE(keywords, '') as keywords 
		FROM posts 
		WHERE meta_description IS NULL OR meta_description = '' OR keywords IS NULL OR keywords = ''
	`)
	if err != nil {
		log.Fatal("Error querying posts:", err)
	}
	defer rows.Close()

	var updatedCount int
	for rows.Next() {
		var post model.Post
		err := rows.Scan(&post.ID, &post.Title, &post.Body, &post.Date, &post.Slug, &post.CreatedAt, &post.UpdatedAt, &post.MetaDescription, &post.Keywords)
		if err != nil {
			log.Printf("Error scanning post %d: %v", post.ID, err)
			continue
		}

		// Generate default SEO fields
		post.GenerateDefaultSEOFields()
		if err := post.ValidateAndSanitizeSEOFields(); err != nil {
			log.Printf("Error validating SEO fields for post %d: %v", post.ID, err)
			continue
		}

		// Update the post with generated SEO fields
		_, err = db.Exec(`
			UPDATE posts 
			SET meta_description = $1, keywords = $2, updated_at = CURRENT_TIMESTAMP 
			WHERE id = $3
		`, post.MetaDescription, post.Keywords, post.ID)
		if err != nil {
			log.Printf("Error updating post %d: %v", post.ID, err)
			continue
		}

		log.Printf("Updated SEO fields for post %d: '%s'", post.ID, post.Title)
		updatedCount++
	}

	log.Printf("Successfully updated SEO fields for %d posts", updatedCount)
}
