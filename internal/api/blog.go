package api

import (
	"errors"
	"lightgo/internal/model"
	"lightgo/internal/store"
	"lightgo/lightgo"
	"net/http"
	"strconv"
)

func queryInt(c *lightgo.Context, name string, fallback int) int {
	value, err := strconv.Atoi(c.Query(name))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
func (a *API) ListPosts(c *lightgo.Context) {
	u, _ := CurrentUser(c)
	page := a.Store.ListPosts(model.PostFilter{
		Page: queryInt(c, "page", 1), PageSize: queryInt(c, "pageSize", 10),
		Keyword: c.Query("keyword"), Category: c.Query("category"), Status: c.Query("status"),
		Sort: c.QueryDefault("sort", "created"), ViewerID: u.ID,
	})
	Success(c, http.StatusOK, page)
}
func (a *API) GetPost(c *lightgo.Context) {
	id, err := store.ParseID(c.Param("id"))
	if err != nil {
		Failure(c, http.StatusBadRequest, "文章 ID 无效", nil)
		return
	}
	u, _ := CurrentUser(c)
	post, err := a.Store.Post(id, u.ID, true)
	if errors.Is(err, store.ErrNotFound) {
		Failure(c, http.StatusNotFound, "文章不存在", nil)
		return
	}
	if err != nil {
		_ = c.Error(err)
		return
	}
	Success(c, http.StatusOK, post)
}
func (a *API) CreatePost(c *lightgo.Context) {
	u, ok := CurrentUser(c)
	if !ok {
		Failure(c, http.StatusUnauthorized, "请先登录", nil)
		return
	}
	var req model.PostRequest
	if !bindRequest(c, &req) {
		return
	}
	post, err := a.Store.CreatePost(u.ID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	Success(c, http.StatusCreated, post)
}
func (a *API) UpdatePost(c *lightgo.Context) {
	u, ok := CurrentUser(c)
	if !ok {
		Failure(c, http.StatusUnauthorized, "请先登录", nil)
		return
	}
	id, err := store.ParseID(c.Param("id"))
	if err != nil {
		Failure(c, http.StatusBadRequest, "文章 ID 无效", nil)
		return
	}
	var req model.PostRequest
	if !bindRequest(c, &req) {
		return
	}
	post, err := a.Store.UpdatePost(id, u.ID, u.Role == "admin", req)
	if errors.Is(err, store.ErrNotFound) {
		Failure(c, http.StatusNotFound, "文章不存在", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		Failure(c, http.StatusForbidden, "无权编辑此文章", nil)
		return
	}
	if err != nil {
		_ = c.Error(err)
		return
	}
	Success(c, http.StatusOK, post)
}
func (a *API) DeletePost(c *lightgo.Context) {
	u, ok := CurrentUser(c)
	if !ok {
		Failure(c, http.StatusUnauthorized, "请先登录", nil)
		return
	}
	id, err := store.ParseID(c.Param("id"))
	if err != nil {
		Failure(c, http.StatusBadRequest, "文章 ID 无效", nil)
		return
	}
	err = a.Store.DeletePost(id, u.ID, u.Role == "admin")
	if errors.Is(err, store.ErrNotFound) {
		Failure(c, http.StatusNotFound, "文章不存在", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		Failure(c, http.StatusForbidden, "无权删除此文章", nil)
		return
	}
	if err != nil {
		_ = c.Error(err)
		return
	}
	Success(c, http.StatusOK, map[string]any{"deleted": id})
}
