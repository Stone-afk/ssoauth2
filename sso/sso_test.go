package sso

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"html/template"
	"net/http"
	"net/url"
	"ssoauth2/web"
	"ssoauth2/web/context"
	webTlp "ssoauth2/web/template"
	"strings"
	"testing"
	"time"
)

var SSOSessionStore = make(map[string]any)

var whiteList = map[string]string{
	"app1": "localhost:8081",
	"app2": "localhost:8082",
}

var bizRedirectUrl = map[string]string{
	"app1": "http://app1.com:8081/token?token=%s",
	"app2": "http://app2.com:8082/token?token=%s",
}

func TestSSOServer(t *testing.T) {
	tpls := template.New("test_server")
	tpls, err := tpls.ParseGlob("./template/*")
	if err != nil {
		t.Fatal(err)
	}
	engine := &webTlp.GoTemplateEngine{
		T: tpls,
	}
	server := web.NewHTTPServer(web.ServerWithTemplateEngine(engine))
	tokens := cache.New(time.Minute*3, time.Minute)
	sessCache := cache.New(time.Minute*15, time.Minute)
	server.Post("/hello", func(ctx *context.Context) {
		_ = ctx.RespString(http.StatusOK, "欢迎来到 SSO")
	})

	server.Post("/logout", func(ctx *context.Context) {
		ck, err := ctx.Request.Cookie("sso_ssid")
		if err != nil {
			_ = ctx.RespString(http.StatusUnauthorized, "请登录")
			return
		}
		ssid := ck.Value
		sessCache.Delete(ssid)
		ck = &http.Cookie{
			Name:   "sso_ssid",
			Value:  ssid,
			MaxAge: -1,
			// 在 https 里面才能用这个 cookie
			//Secure: true,
			// 前端没有办法通过 JS 来访问 cookie
			HttpOnly: true,
		}
		// 强制删除 cookie
		ctx.SetCookie(ck)
		_ = ctx.RespString(http.StatusOK, "退出登录成功")
	})

	server.Post("/login", func(ctx *context.Context) {
		email, _ := ctx.FormValue("email").String()
		pwd, _ := ctx.FormValue("password").String()

		// 这个地方怎么办？是不是要跳回去？
		path, err := ctx.FormValue("redirect_uri").String()
		if err != nil {
			_ = ctx.RespString(http.StatusBadRequest, "登录失败")
			return
		}
		appId, err := ctx.FormValue("app_id").String()
		if err != nil {
			_ = ctx.RespString(http.StatusBadRequest, "登录失败")
			return
		}
		// redirect_uri 必须是某个白名单里面的域名
		decodePath, err := url.PathUnescape(path)
		if err != nil {
			_ = ctx.RespString(http.StatusBadRequest, "登录失败")
			return
		}
		target, ok := whiteList[appId]
		if !ok {
			_ = ctx.RespString(http.StatusBadRequest, "登录失败")
			return
		}
		//url.Parse()
		if !strings.HasPrefix(decodePath, "http:"+target) &&
			!strings.HasPrefix(decodePath, "https:"+target) {
			_ = ctx.RespString(http.StatusBadRequest, "登录失败")
			return
		}
		// 再去查询数据库
		if email == "123@qq.com" && pwd == "123456" {
			ssid := uuid.New().String()
			// 这边要怎么办？
			// 在这边你要设置好 session
			ck := &http.Cookie{
				Name:   "sso_ssid",
				Value:  ssid,
				MaxAge: 1800,
				// 在 https 里面才能用这个 cookie
				//Secure: true,
				// 前端没有办法通过 JS 来访问 cookie
				HttpOnly: true,
			}
			sessCache.Set(ssid, Session{Uid: 123}, time.Minute*15)
			ctx.SetCookie(ck)
			// 带上一个 token，这时候你就要考虑，怎么生成 token？
			// 这里我假设，你的 token 就是一个 uuid，然后你本地有一个 uuid 列表，
			token := uuid.New().String()
			tokens.Set(token, appId, time.Minute)
			tokenUrl, _ := bizRedirectUrl[appId]
			ctx.Redirect(tokenUrl + fmt.Sprintf("?redirect_uri=%s&token=%s", path, token))
			return
		}
		_ = ctx.RespString(http.StatusBadRequest, "登录失败")
		return
	})

	// 这个地方怎么写？
	// 要有一个新的 HTTP 接口
	// 要判断登录态，如果没登录就返回登录页面，
	// 如果登录了，就跳转回去 A/B
	// 这边主要是安全性问题
	server.Post("/check_login", func(ctx *context.Context) {

		// 白名单校验提前到这里
		val := ctx.FormValue("redirect_uri")
		path, err := val.String()
		if err != nil {
			_ = ctx.RespString(http.StatusBadRequest, "登录失败")
			return
		}
		val = ctx.FormValue("app_id")
		appId, err := val.String()
		if err != nil {
			_ = ctx.RespString(http.StatusBadRequest, "登录失败")
			return
		}
		// redirect_uri 必须是某个白名单里面的域名
		decodePath, err := url.PathUnescape(path)
		if err != nil {
			_ = ctx.RespString(http.StatusBadRequest, "登录失败")
			return
		}
		target, ok := whiteList[appId]
		if !ok {
			_ = ctx.RespString(http.StatusBadRequest, "登录失败")
			return
		}

		if !strings.HasPrefix(decodePath, "http:"+target) &&
			!strings.HasPrefix(decodePath, "https:"+target) {
			_ = ctx.RespString(http.StatusBadRequest, "登录失败")
			return
		}

		ck, err := ctx.Request.Cookie("sso_ssid")
		val = ctx.QueryValue("redirect_uri")
		path, _ = val.String()
		path, _ = url.PathUnescape(path)
		if err != nil {
			_ = ctx.Render("login.gohtml", map[string]string{
				"RedirectURI": path,
			})
			return
		}

		// 尽可能在这一句之前，过滤掉非法请求
		ssid := ck.Value
		_, ok = sessCache.Get(ssid)
		if !ok {
			_ = ctx.Render("login.gohtml", map[string]string{
				"RedirectURI": path,
			})
			return
		}
		// 这边就是登录了
		// 要跳回去
		token := uuid.New().String()
		tokens.Set(token, appId, time.Minute)
		tokenUrl, _ := bizRedirectUrl[appId]
		ctx.Redirect(tokenUrl + fmt.Sprintf("?redirect_uri=%s&token=%s", path, token))
	})

	// token 校验，保护好
	// 请求来源可以要求一个 app 一个 IP
	server.Post("/token/validate", func(ctx *context.Context) {
		token, _ := ctx.QueryValue("token").String()
		// 可能会有一个解密的过程
		_, ok := tokens.Get(token)
		if !ok {
			_ = ctx.RespString(http.StatusForbidden, "没有权限")
			return
		}
		// 带上用户信息，比如说 uid
		_ = ctx.RespString(http.StatusOK, "123")
	})

	_ = server.Start(":8083")
}

type Session struct {
	// session 里面放的内容，就是 UID，你有需要你可以继续加
	Uid uint64
}
