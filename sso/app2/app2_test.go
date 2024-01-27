package app2

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"net/url"
	"ssoauth2/web/context"
	webHandler "ssoauth2/web/handler"
	"testing"
	"time"
)

var sessCache = cache.New(time.Minute*15, time.Minute)

func TestApp2Server(t *testing.T) {

}

func app2LoginMiddleware(next webHandler.HandleFunc) webHandler.HandleFunc {
	return func(ctx *context.Context) {
		if ctx.Request.URL.Path == "/login" || ctx.Request.URL.Path == "/health" {
			next(ctx)
			return
		}
		path := ctx.Request.URL.String()
		// sso 链接
		const pattern = "http://localhost:8083/check_login?redirect_uri=%sapp_id=%s"
		// URL 编码
		path = fmt.Sprintf(pattern, url.PathEscape(path), "app2")
		ck, err := ctx.Request.Cookie("app2_ssid")
		if err != nil {
			// 这个地方要考虑跳转，跳过去 SSO 里面
			ctx.Redirect(path)
			//ctx.RespString(http.StatusUnauthorized, "请登录")
			return
		}
		ssid := ck.Value
		sess, ok := sessCache.Get(ssid)
		if !ok {
			ctx.Redirect(path)
			//ctx.RespString(http.StatusUnauthorized, "请登录")
			return
		}
		ctx.UserValues["sess"] = sess
		next(ctx)
	}
}