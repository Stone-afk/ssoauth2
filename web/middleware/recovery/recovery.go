package recovery

import (
	"log"
	"net/http"
	"web/context"
	webHandler "web/handler"
	"web/middleware"
)

func (m *MiddlewareBuilder) Build() middleware.Middleware {
	return func(next webHandler.HandleFunc) webHandler.HandleFunc {
		return func(ctx *context.Context) {
			defer func() {
				if err := recover(); err != nil {
					ctx.RespStatusCode = m.StatusCode
					ctx.RespData = []byte(m.ErrMsg)
					// 万一 LogFunc 也panic，那我们也无能为力了
					m.LogFunc(ctx)
				}
			}()
			// 这里就是before route, before execute
			next(ctx)
			// 这里就是after route, after execute
		}
	}
}

func NewBuilder() *MiddlewareBuilder {
	return &MiddlewareBuilder{
		StatusCode: http.StatusInternalServerError,
		ErrMsg:     "服务器未知错误，请联系管理员!",
		LogFunc: func(ctx *context.Context) {
			log.Println(string(ctx.RespData))
		},
	}
}

type MiddlewareBuilder struct {
	StatusCode int
	ErrMsg     string
	LogFunc    func(ctx *context.Context)
}
