package opentelemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"web/context"
	webHandler "web/handler"
	"web/middleware"
)

const defaultInstrumentationName = "go/web/middle/opentelemetry"

type MiddlewareBuilder struct {
	Tracer trace.Tracer
}

func NewBuilder() *MiddlewareBuilder {
	return &MiddlewareBuilder{}
}

func (b *MiddlewareBuilder) Build() middleware.Middleware {
	if b.Tracer == nil {
		b.Tracer = otel.GetTracerProvider().Tracer(defaultInstrumentationName)
	}
	initZipkin()
	return func(next webHandler.HandleFunc) webHandler.HandleFunc {
		return func(ctx *context.Context) {
			// 为了和上游链路连在一起，也就是发起 HTTP 请求的客户端 (关联上下游)
			reqCtx := ctx.Request.Context()
			reqCtx = otel.GetTextMapPropagator().Extract(reqCtx,
				propagation.HeaderCarrier(ctx.Request.Header),
			)
			//  设置各种值
			reqCtx, span := b.Tracer.Start(reqCtx, "unknown", trace.WithAttributes())
			span.SetAttributes(attribute.String("http.method", ctx.Request.Method))
			span.SetAttributes(attribute.String("peer.hostname", ctx.Request.Host))
			span.SetAttributes(attribute.String("http.url", ctx.Request.URL.String()))
			span.SetAttributes(attribute.String("http.scheme", ctx.Request.URL.Scheme))
			span.SetAttributes(attribute.String("span.kind", "server"))
			span.SetAttributes(attribute.String("component", "web"))
			span.SetAttributes(attribute.String("peer.address", ctx.Request.RemoteAddr))
			span.SetAttributes(attribute.String("http.proto", ctx.Request.Proto))

			// span.End 执行之后，就意味着 span 本身已经确定无疑了，将不能再变化了
			defer span.End()
			// 将 带有 链路追踪信息的 reqCtx 设置回request
			ctx.Request = ctx.Request.WithContext(reqCtx)
			next(ctx)

			// 使用命中的路由来作为 span 的名字
			if ctx.MatchedRoute != "" {
				span.SetName(ctx.MatchedRoute)
			}

			// 怎么拿到响应的状态呢？比如说用户有没有返回错误，响应码是多少，怎么办？
			span.SetAttributes(attribute.Int("http.status", ctx.RespStatusCode))
		}
	}
}
