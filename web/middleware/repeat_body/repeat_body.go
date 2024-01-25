package repeat_body

import (
	"io/ioutil"
	"web/context"
	webHandler "web/handler"
	"web/middleware"
)

func Middleware() middleware.Middleware {
	return func(next webHandler.HandleFunc) webHandler.HandleFunc {
		return func(ctx *context.Context) {
			ctx.Request.Body = ioutil.NopCloser(ctx.Request.Body)
			next(ctx)
		}
	}
}
