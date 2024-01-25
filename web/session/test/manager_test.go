package test

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
	"web/session"
	"web/session/cookie"
	"web/session/memory"
	web "web/v9"
)

func LoginMiddleware() web.Middleware {
	return func(next web.HandleFunc) web.HandleFunc {
		return func(ctx *web.Context) {
			// 是登录请求则执行校验
			if ctx.Request.URL.Path != "/login" {
				m := session.GetManager(ctx)
				sess, err := m.GetSession(ctx)
				// 不管发生了什么错误，对于用户我们都是返回未授权
				if err != nil {
					ctx.RespStatusCode = http.StatusUnauthorized
					return
				}
				// 每次收到一个请求都刷新
				ctx.UserValues["sess"] = sess
				_ = m.Refresh(ctx.Request.Context(), sess.ID())
				next(ctx)
			}
		}
	}
}

func TestManager(t *testing.T) {
	s := web.NewHTTPServer()

	m := &session.Manager{
		SessCtxKey: "_sess",
		Store:      memory.NewStore(30 * time.Minute),
		Propagator: cookie.NewPropagator("sessid",
			cookie.WithCookieOption(func(c *http.Cookie) {
				c.HttpOnly = true
			})),
	}

	s.Use(session.RegisterManager(m), LoginMiddleware())

	s.Post("/login", func(ctx *web.Context) {
		// 前面就是你登录的时候一大堆的登录校验
		id := uuid.New()
		m := session.GetManager(ctx)
		sess, err := m.InitSession(ctx, id.String())
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError
			return
		}
		// 然后根据自己的需要设置
		err = sess.Set(ctx.Request.Context(), "mykey", "some value")
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError
			return
		}
	})
	s.Get("/resource", func(ctx *web.Context) {
		m := session.GetManager(ctx)
		sess, err := m.GetSession(ctx)
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError
			return
		}
		val, err := sess.Get(ctx.Request.Context(), "mykey")
		ctx.RespData = []byte(val)
	})

	s.Post("/logout", func(ctx *web.Context) {
		m := session.GetManager(ctx)
		_ = m.RemoveSession(ctx)
	})

	err := s.Start(":8081")
	assert.Nil(t, err)
}
