package store

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"lightgo/internal/model"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound  = errors.New("resource not found")
	ErrConflict  = errors.New("resource already exists")
	ErrForbidden = errors.New("operation forbidden")
)

type Store struct {
	mu          sync.RWMutex
	users       map[int64]model.User
	byName      map[string]int64
	posts       map[int64]model.Post
	comments    map[int64]model.Comment
	tokens      map[string]model.Token
	nextUser    int64
	nextPost    int64
	nextComment int64
	secret      []byte
}

func New() *Store {
	s := &Store{
		users:    make(map[int64]model.User),
		byName:   make(map[string]int64),
		posts:    make(map[int64]model.Post),
		comments: make(map[int64]model.Comment),
		tokens:   make(map[string]model.Token),
		secret:   make([]byte, 32),
	}
	if _, err := rand.Read(s.secret); err != nil {
		s.secret = []byte("lightgo-development-secret-key")
	}
	s.Seed()
	return s
}
func hashPassword(username, password string) string {
	sum := sha256.Sum256([]byte("lightgo:" + strings.ToLower(username) + ":" + password))
	return fmt.Sprintf("%x", sum[:])
}
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users = make(map[int64]model.User)
	s.byName = make(map[string]int64)
	s.posts = make(map[int64]model.Post)
	s.comments = make(map[int64]model.Comment)
	s.tokens = make(map[string]model.Token)
	s.nextUser = 0
	s.nextPost = 0
	s.nextComment = 0
}
func (s *Store) Seed() {
	s.Reset()
	admin, _ := s.CreateUser("admin", "secret123", "admin")
	alice, _ := s.CreateUser("alice", "secret123", "author")
	categories := []string{"Go", "Web", "Engineering", "Tutorial"}
	for i := 1; i <= 8; i++ {
		author := admin.ID
		if i%2 == 0 {
			author = alice.ID
		}
		status := "published"
		if i == 8 {
			status = "draft"
		}
		_, _ = s.CreatePost(author, model.PostRequest{
			Title:    fmt.Sprintf("LightGo 示例文章 %d", i),
			Summary:  "展示路由、中间件、绑定校验、模板渲染与静态文件服务。",
			Content:  "LightGo 只依赖 Go 标准库，适合学习 HTTP 框架的核心实现。\n\n本文是内置演示数据。",
			Category: categories[(i-1)%len(categories)],
			Tags:     []string{"Go", "LightGo"}, Status: status,
		})
	}
	_, _ = s.CreateComment(1, admin.ID, model.CommentRequest{Content: "欢迎来到 LightGo，欢迎交流框架实现思路。"})
	_, _ = s.CreateComment(1, alice.ID, model.CommentRequest{Content: "标准库也能完成一个结构清晰的 Web 演示站。"})
	_, _ = s.CreateComment(2, admin.ID, model.CommentRequest{Content: "文章路由和中间件链是理解框架的好起点。"})
}
func (s *Store) CreateUser(username, password, role string) (model.User, error) {
	username = strings.TrimSpace(username)
	key := strings.ToLower(username)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byName[key]; ok {
		return model.User{}, ErrConflict
	}
	s.nextUser++
	u := model.User{ID: s.nextUser, Username: username, PasswordHash: hashPassword(username, password), Role: role, CreatedAt: time.Now()}
	s.users[u.ID] = u
	s.byName[key] = u.ID
	return u, nil
}
func (s *Store) Authenticate(username, password string) (model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byName[strings.ToLower(strings.TrimSpace(username))]
	if !ok {
		return model.User{}, ErrNotFound
	}
	u := s.users[id]
	if !hmac.Equal([]byte(u.PasswordHash), []byte(hashPassword(u.Username, password))) {
		return model.User{}, ErrNotFound
	}
	return u, nil
}
func (s *Store) User(id int64) (model.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	return u, ok
}
func (s *Store) Users() []model.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func (s *Store) IssueToken(userID int64, ttl time.Duration) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return "", ErrNotFound
	}
	expires := time.Now().Add(ttl).Unix()
	nonce := make([]byte, 12)
	_, _ = rand.Read(nonce)
	payload := fmt.Sprintf("%d.%d.%s", userID, expires, base64.RawURLEncoding.EncodeToString(nonce))
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(payload))
	value := payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	s.tokens[value] = model.Token{Value: value, UserID: userID, ExpiresAt: time.Unix(expires, 0)}
	return value, nil
}
func (s *Store) ValidateToken(value string) (model.User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tokens[value]
	if !ok || time.Now().After(t.ExpiresAt) {
		delete(s.tokens, value)
		return model.User{}, false
	}
	u, ok := s.users[t.UserID]
	return u, ok
}
func (s *Store) CreatePost(authorID int64, req model.PostRequest) (model.Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[authorID]; !ok {
		return model.Post{}, ErrNotFound
	}
	s.nextPost++
	now := time.Now()
	p := model.Post{ID: s.nextPost, Title: strings.TrimSpace(req.Title), Summary: strings.TrimSpace(req.Summary), Content: req.Content, AuthorID: authorID, Category: strings.TrimSpace(req.Category), Tags: append([]string(nil), req.Tags...), Status: req.Status, CreatedAt: now, UpdatedAt: now}
	s.posts[p.ID] = p
	return s.withAuthorLocked(p), nil
}
func (s *Store) Post(id, viewerID int64, increment bool) (model.Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.posts[id]
	if !ok || (p.Status == "draft" && p.AuthorID != viewerID) {
		return model.Post{}, ErrNotFound
	}
	if increment {
		p.Views++
		s.posts[id] = p
	}
	return s.withAuthorLocked(p), nil
}
func (s *Store) UpdatePost(id, actorID int64, admin bool, req model.PostRequest) (model.Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.posts[id]
	if !ok {
		return model.Post{}, ErrNotFound
	}
	if p.AuthorID != actorID && !admin {
		return model.Post{}, ErrForbidden
	}
	p.Title, p.Summary, p.Content = strings.TrimSpace(req.Title), strings.TrimSpace(req.Summary), req.Content
	p.Category, p.Tags, p.Status = strings.TrimSpace(req.Category), append([]string(nil), req.Tags...), req.Status
	p.UpdatedAt = time.Now()
	s.posts[id] = p
	return s.withAuthorLocked(p), nil
}
func (s *Store) DeletePost(id, actorID int64, admin bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.posts[id]
	if !ok {
		return ErrNotFound
	}
	if p.AuthorID != actorID && !admin {
		return ErrForbidden
	}
	delete(s.posts, id)
	for commentID, comment := range s.comments {
		if comment.PostID == id {
			delete(s.comments, commentID)
		}
	}
	return nil
}
func (s *Store) ListPosts(f model.PostFilter) model.Page[model.Post] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f.Page = max(f.Page, 1)
	if f.PageSize < 1 {
		f.PageSize = 10
	}
	f.PageSize = min(f.PageSize, 100)
	needle := strings.ToLower(strings.TrimSpace(f.Keyword))
	items := make([]model.Post, 0)
	for _, p := range s.posts {
		if p.Status == "draft" && p.AuthorID != f.ViewerID {
			continue
		}
		if f.Status != "" && p.Status != f.Status {
			continue
		}
		if f.Category != "" && !strings.EqualFold(p.Category, f.Category) {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(p.Title+" "+p.Summary), needle) {
			continue
		}
		items = append(items, s.withAuthorLocked(p))
	}
	sort.Slice(items, func(i, j int) bool {
		if f.Sort == "views" {
			return items[i].Views > items[j].Views
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	total := len(items)
	start := min((f.Page-1)*f.PageSize, total)
	end := min(start+f.PageSize, total)
	pages := (total + f.PageSize - 1) / f.PageSize
	return model.Page[model.Post]{Items: items[start:end], Page: f.Page, PageSize: f.PageSize, Total: total, Pages: pages}
}
func (s *Store) Stats() model.Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	posts := make([]model.Post, 0, len(s.posts))
	var views int64
	for _, p := range s.posts {
		views += p.Views
		if p.Status == "published" {
			posts = append(posts, s.withAuthorLocked(p))
		}
	}
	sort.Slice(posts, func(i, j int) bool { return posts[i].CreatedAt.After(posts[j].CreatedAt) })
	posts = posts[:min(len(posts), 5)]
	return model.Stats{Posts: len(s.posts), Users: len(s.users), Views: views, LatestPosts: posts}
}
func (s *Store) withAuthorLocked(p model.Post) model.Post {
	if u, ok := s.users[p.AuthorID]; ok {
		p.Author = u.Username
	}
	p.CommentCount = s.commentCountLocked(p.ID)
	return p
}
func (s *Store) withCommentAuthorLocked(comment model.Comment) model.Comment {
	if u, ok := s.users[comment.AuthorID]; ok {
		comment.Author = u.Username
	}
	return comment
}
func (s *Store) commentCountLocked(postID int64) int {
	count := 0
	for _, comment := range s.comments {
		if comment.PostID == postID {
			count++
		}
	}
	return count
}
func ParseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}
