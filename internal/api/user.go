package api

import (
	"errors"
	"lightgo/internal/model"
	"lightgo/internal/store"
	"lightgo/lightgo"
	"lightgo/lightgo/binding"
	"net/http"
	"strings"
	"time"
)

type API struct{ Store *store.Store }

func New(s *store.Store) *API { return &API{Store: s} }
func Success(c *lightgo.Context, status int, data any) {
	_ = c.JSON(status, map[string]any{"code": 0, "message": "ok", "data": data})
}
func Failure(c *lightgo.Context, status int, message string, data any) {
	_ = c.JSON(status, map[string]any{"code": status, "message": message, "data": data})
}
func bindRequest(c *lightgo.Context, dst any) bool {
	if err := c.Bind(dst); err != nil {
		var validation binding.ValidationErrors
		if errors.As(err, &validation) {
			Failure(c, http.StatusUnprocessableEntity, "参数校验失败", map[string]any{"errors": validation})
		} else {
			Failure(c, http.StatusBadRequest, "请求参数解析失败", map[string]any{"error": err.Error()})
		}
		return false
	}
	return true
}
func (a *API) Register(c *lightgo.Context) {
	var req model.RegisterRequest
	if !bindRequest(c, &req) {
		return
	}
	u, err := a.Store.CreateUser(req.Username, req.Password, "author")
	if errors.Is(err, store.ErrConflict) {
		Failure(c, http.StatusConflict, "用户名已存在", nil)
		return
	}
	if err != nil {
		_ = c.Error(err)
		return
	}
	Success(c, http.StatusCreated, u)
}
func (a *API) Login(c *lightgo.Context) {
	var req model.LoginRequest
	if !bindRequest(c, &req) {
		return
	}
	u, err := a.Store.Authenticate(req.Username, req.Password)
	if err != nil {
		Failure(c, http.StatusUnauthorized, "用户名或密码错误", nil)
		return
	}
	token, err := a.Store.IssueToken(u.ID, 24*time.Hour)
	if err != nil {
		_ = c.Error(err)
		return
	}
	Success(c, http.StatusOK, map[string]any{"token": token, "user": u})
}
func (a *API) Users(c *lightgo.Context) { Success(c, http.StatusOK, a.Store.Users()) }
func (a *API) Stats(c *lightgo.Context) { Success(c, http.StatusOK, a.Store.Stats()) }
func (a *API) Auth(required bool) lightgo.Middleware {
	return func(c *lightgo.Context, next lightgo.NextFunc) {
		raw := strings.TrimSpace(c.Header("Authorization"))
		if raw == "" {
			if required {
				Failure(c, http.StatusUnauthorized, "请先登录", nil)
				c.Abort()
				return
			}
			next()
			return
		}
		parts := strings.Fields(raw)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			if required {
				Failure(c, http.StatusUnauthorized, "Authorization 格式错误", nil)
				c.Abort()
				return
			}
			next()
			return
		}
		u, ok := a.Store.ValidateToken(parts[1])
		if !ok {
			if required {
				Failure(c, http.StatusUnauthorized, "登录凭证无效或已过期", nil)
				c.Abort()
				return
			}
			next()
			return
		}
		c.Set("user", u)
		next()
	}
}
func CurrentUser(c *lightgo.Context) (model.User, bool) {
	value, ok := c.Get("user")
	if !ok {
		return model.User{}, false
	}
	u, ok := value.(model.User)
	return u, ok
}
