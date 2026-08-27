package main

import (
	"lightgo/internal/api"
	"lightgo/internal/model"
	"lightgo/internal/store"
	"lightgo/lightgo"
	"lightgo/lightgo/render"
	staticfiles "lightgo/lightgo/static"
	"net/http"
	"path/filepath"
	"strings"
)

func registerRoutes(engine *lightgo.Engine, data *store.Store, webDir string) error {
	templates := render.NewTemplateEngine()
	if err := templates.Load(filepath.Join(webDir, "templates")); err != nil {
		return err
	}
	engine.SetTemplates(templates)
	handlers := api.New(data)
	engine.GET("/static/*path", staticfiles.FileServer(filepath.Join(webDir, "static"), staticfiles.Options{}))
	engine.GET("/", func(c *lightgo.Context) {
		_ = c.HTML(http.StatusOK, "index", pageData("LightGo", map[string]any{"Stats": data.Stats()}))
	})
	engine.GET("/blog", func(c *lightgo.Context) {
		page := data.ListPosts(model.PostFilter{Page: queryPage(c), PageSize: 6, Keyword: c.Query("keyword"), Category: c.Query("category")})
		_ = c.HTML(http.StatusOK, "blog_list", pageData("文章", map[string]any{"Page": page, "Keyword": c.Query("keyword"), "Category": c.Query("category")}))
	})
	engine.GET("/categories", func(c *lightgo.Context) {
		_ = c.HTML(http.StatusOK, "category_list", pageData("分类", map[string]any{"Categories": data.Categories()}))
	})
	engine.GET("/blog/new", func(c *lightgo.Context) {
		_ = c.HTML(http.StatusOK, "blog_form", pageData("新建文章", map[string]any{"Mode": "create"}))
	})
	engine.GET("/blog/:id/edit", func(c *lightgo.Context) {
		id, err := store.ParseID(c.Param("id"))
		if err != nil {
			renderPageError(c, 400, "文章 ID 无效")
			return
		}
		post, err := data.Post(id, 0, false)
		if err != nil {
			renderPageError(c, 404, "文章不存在")
			return
		}
		_ = c.HTML(http.StatusOK, "blog_form", pageData("编辑文章", map[string]any{"Mode": "edit", "Post": post}))
	})
	engine.GET("/blog/:id", func(c *lightgo.Context) {
		id, err := store.ParseID(c.Param("id"))
		if err != nil {
			renderPageError(c, 400, "文章 ID 无效")
			return
		}
		post, err := data.Post(id, 0, true)
		if err != nil {
			renderPageError(c, 404, "文章不存在")
			return
		}
		comments, err := data.Comments(id, 0)
		if err != nil {
			renderPageError(c, 404, "文章不存在")
			return
		}
		_ = c.HTML(http.StatusOK, "blog_detail", pageData(post.Title, map[string]any{"Post": post, "Comments": comments}))
	})
	engine.GET("/users", func(c *lightgo.Context) {
		_ = c.HTML(http.StatusOK, "user_list", pageData("用户", map[string]any{"Users": data.Users()}))
	})
	engine.GET("/login", func(c *lightgo.Context) {
		_ = c.HTML(http.StatusOK, "login", pageData("登录", nil))
	})
	apiGroup := engine.Group("/api")
	auth := apiGroup.Group("/auth")
	auth.POST("/register", handlers.Register)
	auth.POST("/login", handlers.Login)
	apiGroup.GET("/users", handlers.Users)
	apiGroup.GET("/stats", handlers.Stats)
	apiGroup.GET("/blog/categories", handlers.Categories)
	posts := apiGroup.Group("/blog/posts")
	posts.GET("", handlers.ListPosts, handlers.Auth(false))
	posts.GET("/:id", handlers.GetPost, handlers.Auth(false))
	posts.POST("", handlers.CreatePost, handlers.Auth(true))
	posts.PUT("/:id", handlers.UpdatePost, handlers.Auth(true))
	posts.PATCH("/:id", handlers.UpdatePost, handlers.Auth(true))
	posts.DELETE("/:id", handlers.DeletePost, handlers.Auth(true))
	posts.GET("/:id/comments", handlers.ListComments, handlers.Auth(false))
	posts.POST("/:id/comments", handlers.CreateComment, handlers.Auth(true))
	posts.DELETE("/:id/comments/:commentID", handlers.DeleteComment, handlers.Auth(true))
	engine.NotFound = func(c *lightgo.Context) {
		if acceptsHTML(c.Request) {
			renderPageError(c, 404, "页面不存在")
			return
		}
		api.Failure(c, 404, "资源不存在", nil)
	}
	return nil
}
func pageData(title string, values map[string]any) map[string]any {
	data := map[string]any{"Title": title}
	for key, value := range values {
		data[key] = value
	}
	return data
}
func renderPageError(c *lightgo.Context, status int, message string) {
	_ = c.HTML(status, "error", pageData(http.StatusText(status), map[string]any{"Status": status, "Message": message}))
}
func acceptsHTML(r *http.Request) bool {
	return r.Header.Get("Accept") == "" || r.Header.Get("Accept") == "*/*" || strings.Contains(r.Header.Get("Accept"), "text/html")
}
func queryPage(c *lightgo.Context) int {
	var query struct {
		Page int `query:"page" default:"1"`
	}
	if err := c.BindQuery(&query); err != nil {
		return 1
	}
	return query.Page
}
