package accesslog

import (
	"testing"
	"time"
	web "web/v8"
)

func TestMiddlewareBuilder_Build(t *testing.T) {
	logbd := NewBuilder()
	s := web.NewHTTPServer()
	s.Get("/", func(ctx *web.Context) {
		ctx.Response.Write([]byte("hello, world"))
	})

	s.Get("/user", func(ctx *web.Context) {
		time.Sleep(time.Second)
		ctx.RespData = []byte("hello, user")
	})
	s.Use(logbd.Build())
	s.Start(":8081")
}
