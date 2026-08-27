package store

import (
	"lightgo/internal/model"
	"sort"
	"strings"
	"time"
)

// Categories returns published-post counts so the public category directory never exposes drafts.
func (s *Store) Categories() []model.Category {
	s.mu.RLock()
	defer s.mu.RUnlock()
	counts := make(map[string]int)
	for _, post := range s.posts {
		if post.Status == "published" {
			counts[post.Category]++
		}
	}
	categories := make([]model.Category, 0, len(counts))
	for name, count := range counts {
		categories = append(categories, model.Category{Name: name, PostCount: count})
	}
	sort.Slice(categories, func(i, j int) bool {
		if categories[i].PostCount == categories[j].PostCount {
			return strings.ToLower(categories[i].Name) < strings.ToLower(categories[j].Name)
		}
		return categories[i].PostCount > categories[j].PostCount
	})
	return categories
}

func (s *Store) Comments(postID, viewerID int64) ([]model.Comment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	post, ok := s.posts[postID]
	if !ok || (post.Status == "draft" && post.AuthorID != viewerID) {
		return nil, ErrNotFound
	}
	comments := make([]model.Comment, 0)
	for _, comment := range s.comments {
		if comment.PostID == postID {
			comments = append(comments, s.withCommentAuthorLocked(comment))
		}
	}
	sort.Slice(comments, func(i, j int) bool { return comments[i].CreatedAt.Before(comments[j].CreatedAt) })
	return comments, nil
}

func (s *Store) CreateComment(postID, authorID int64, req model.CommentRequest) (model.Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	post, ok := s.posts[postID]
	if !ok || post.Status != "published" {
		return model.Comment{}, ErrNotFound
	}
	if _, ok := s.users[authorID]; !ok {
		return model.Comment{}, ErrNotFound
	}
	s.nextComment++
	comment := model.Comment{
		ID: s.nextComment, PostID: postID, AuthorID: authorID,
		Content: strings.TrimSpace(req.Content), CreatedAt: time.Now(),
	}
	s.comments[comment.ID] = comment
	return s.withCommentAuthorLocked(comment), nil
}

func (s *Store) DeleteComment(postID, commentID, actorID int64, admin bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.posts[postID]; !ok {
		return ErrNotFound
	}
	comment, ok := s.comments[commentID]
	if !ok || comment.PostID != postID {
		return ErrNotFound
	}
	if comment.AuthorID != actorID && !admin {
		return ErrForbidden
	}
	delete(s.comments, commentID)
	return nil
}
