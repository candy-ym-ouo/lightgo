package api

import (
	"errors"
	"lightgo/internal/model"
	"lightgo/internal/store"
	"lightgo/lightgo"
	"net/http"
)

func (a *API) Categories(c *lightgo.Context) {
	Success(c, http.StatusOK, a.Store.Categories())
}

func (a *API) ListComments(c *lightgo.Context) {
	postID, err := store.ParseID(c.Param("id"))
	if err != nil {
		Failure(c, http.StatusBadRequest, "文章 ID 无效", nil)
		return
	}
	user, _ := CurrentUser(c)
	comments, err := a.Store.Comments(postID, user.ID)
	if errors.Is(err, store.ErrNotFound) {
		Failure(c, http.StatusNotFound, "文章不存在", nil)
		return
	}
	if err != nil {
		_ = c.Error(err)
		return
	}
	Success(c, http.StatusOK, comments)
}

func (a *API) CreateComment(c *lightgo.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		Failure(c, http.StatusUnauthorized, "请先登录", nil)
		return
	}
	postID, err := store.ParseID(c.Param("id"))
	if err != nil {
		Failure(c, http.StatusBadRequest, "文章 ID 无效", nil)
		return
	}
	var req model.CommentRequest
	if !bindRequest(c, &req) {
		return
	}
	comment, err := a.Store.CreateComment(postID, user.ID, req)
	if errors.Is(err, store.ErrNotFound) {
		Failure(c, http.StatusNotFound, "文章不存在或暂不支持评论", nil)
		return
	}
	if err != nil {
		_ = c.Error(err)
		return
	}
	Success(c, http.StatusCreated, comment)
}

func (a *API) DeleteComment(c *lightgo.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		Failure(c, http.StatusUnauthorized, "请先登录", nil)
		return
	}
	postID, err := store.ParseID(c.Param("id"))
	if err != nil {
		Failure(c, http.StatusBadRequest, "文章 ID 无效", nil)
		return
	}
	commentID, err := store.ParseID(c.Param("commentID"))
	if err != nil {
		Failure(c, http.StatusBadRequest, "评论 ID 无效", nil)
		return
	}
	err = a.Store.DeleteComment(postID, commentID, user.ID, user.Role == "admin")
	if errors.Is(err, store.ErrNotFound) {
		Failure(c, http.StatusNotFound, "评论不存在", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		Failure(c, http.StatusForbidden, "无权删除此评论", nil)
		return
	}
	if err != nil {
		_ = c.Error(err)
		return
	}
	Success(c, http.StatusOK, map[string]any{"deleted": commentID})
}
