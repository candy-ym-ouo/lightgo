package model

import "time"

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
}
type Post struct {
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	Summary      string    `json:"summary"`
	Content      string    `json:"content"`
	AuthorID     int64     `json:"authorId"`
	Author       string    `json:"author,omitempty"`
	Category     string    `json:"category"`
	Tags         []string  `json:"tags"`
	Status       string    `json:"status"`
	Views        int64     `json:"views"`
	CommentCount int       `json:"commentCount"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
type Category struct {
	Name      string `json:"name"`
	PostCount int    `json:"postCount"`
}
type Comment struct {
	ID        int64     `json:"id"`
	PostID    int64     `json:"postId"`
	AuthorID  int64     `json:"authorId"`
	Author    string    `json:"author,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}
type CommentRequest struct {
	Content string `json:"content" validate:"required|min=3|max=500"`
}
type Token struct {
	Value     string
	UserID    int64
	ExpiresAt time.Time
}
type RegisterRequest struct {
	Username string `json:"username" validate:"required|min=3|max=30|alpha"`
	Password string `json:"password" validate:"required|min=6|max=72"`
}
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}
type PostRequest struct {
	Title    string   `json:"title" validate:"required|min=3|max=100"`
	Summary  string   `json:"summary" validate:"required|max=240"`
	Content  string   `json:"content" validate:"required|min=3"`
	Category string   `json:"category" validate:"required"`
	Tags     []string `json:"tags"`
	Status   string   `json:"status" default:"draft" validate:"required|oneof=draft published"`
}
type PostFilter struct {
	Page     int
	PageSize int
	Keyword  string
	Category string
	Status   string
	Sort     string
	ViewerID int64
}
type Page[T any] struct {
	Items    []T `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
	Pages    int `json:"pages"`
}
type Stats struct {
	Posts       int    `json:"posts"`
	Users       int    `json:"users"`
	Views       int64  `json:"views"`
	LatestPosts []Post `json:"latestPosts"`
}
