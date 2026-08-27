package store

import (
	"sync"
	"testing"
	"time"

	"lightgo/internal/model"
)

func TestStoreAuthCRUDAndPermissions(t *testing.T) {
	s := New()
	u, err := s.Authenticate("alice", "secret123")
	if err != nil {
		t.Fatal(err)
	}
	token, err := s.IssueToken(u.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := s.ValidateToken(token); !ok || got.ID != u.ID {
		t.Fatalf("token invalid: %+v %v", got, ok)
	}
	p, err := s.CreatePost(u.ID, model.PostRequest{Title: "A valid title", Summary: "summary", Content: "content", Category: "Go", Status: "published"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdatePost(p.ID, 1, false, model.PostRequest{Title: "updated", Summary: "s", Content: "c", Category: "Go", Status: "draft"}); err != ErrForbidden {
		t.Fatalf("err=%v", err)
	}
	if err := s.DeletePost(p.ID, u.ID, false); err != nil {
		t.Fatal(err)
	}
}

func TestStoreConcurrentReads(t *testing.T) {
	s := New()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_ = s.Stats()
				_ = s.Users()
				_ = s.ListPosts(model.PostFilter{})
			}
		}()
	}
	wg.Wait()
}

func TestStoreCommentsAndCategories(t *testing.T) {
	s := New()
	alice, err := s.Authenticate("alice", "secret123")
	if err != nil {
		t.Fatal(err)
	}
	comment, err := s.CreateComment(1, alice.ID, model.CommentRequest{Content: "一条新的测试评论"})
	if err != nil {
		t.Fatal(err)
	}
	comments, err := s.Comments(1, 0)
	if err != nil || len(comments) != 3 {
		t.Fatalf("comments=%d err=%v", len(comments), err)
	}
	if err := s.DeleteComment(1, comment.ID, 1, false); err != ErrForbidden {
		t.Fatalf("permission err=%v", err)
	}
	if err := s.DeleteComment(1, comment.ID, alice.ID, false); err != nil {
		t.Fatal(err)
	}
	categories := s.Categories()
	if len(categories) != 4 || categories[0].PostCount != 2 {
		t.Fatalf("categories=%+v", categories)
	}
}
