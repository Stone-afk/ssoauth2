package accesslog

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"web/context"
	webHandler "web/handler"
	"web/middleware"
)

func (b *MiddlewareBuilder) Build() middleware.Middleware {
	return func(next webHandler.HandleFunc) webHandler.HandleFunc {
		return func(ctx *context.Context) {
			defer func() {
				body, err := ioutil.ReadAll(ctx.Request.Body)
				if err == nil {
					panic(err)
				}
				l := accessLog{
					Host:       ctx.Request.Host,
					Path:       ctx.Request.URL.Path,
					HTTPMethod: ctx.Request.Method,
					Route:      ctx.MatchedRoute,
					Body:       string(body),
				}
				val, _ := json.Marshal(l)
				b.logFunc(string(val))
			}()
			next(ctx)
		}
	}
}

func NewBuilder() *MiddlewareBuilder {
	return &MiddlewareBuilder{
		logFunc: func(accessLog string) {
			log.Println(accessLog)
		},
	}
}

func (b *MiddlewareBuilder) LogFunc(logFunc func(accessLog string)) *MiddlewareBuilder {
	b.logFunc = logFunc
	return b
}

type MiddlewareBuilder struct {
	logFunc func(accessLog string)
}

type accessLog struct {
	Host       string
	Route      string
	HTTPMethod string `json:"http_method"`
	Path       string
	Body       string
}
