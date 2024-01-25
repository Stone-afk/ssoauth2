package opentelemetry

import (
	"go.opentelemetry.io/otel"
	"testing"
	"time"
	"web"
	"web/context"
)

func TestMiddlewareBuilder_Build(t *testing.T) {
	tracer := otel.GetTracerProvider().Tracer("")
	testinitZipkin(t)
	s := web.NewHTTPServer()

	s.Get("/", func(ctx *context.Context) {
		ctx.Response.Write([]byte("hello, world"))
	})

	s.Get("/order", func(ctx *context.Context) {
		c, span := tracer.Start(ctx.Request.Context(), "first_layer")
		defer span.End()

		c, second := tracer.Start(c, "second_layer")
		time.Sleep(time.Second)

		c, third1 := tracer.Start(c, "third_layer_1")
		time.Sleep(100 * time.Millisecond)
		third1.End()
		c, third2 := tracer.Start(c, "third_layer_1")
		time.Sleep(300 * time.Millisecond)
		third2.End()
		second.End()
		ctx.RespStatusCode = 200
		ctx.RespData = []byte("hello, world")
	})

	s.Get("/user", func(ctx *context.Context) {
		c, span := tracer.Start(ctx.Request.Context(), "first_layer")
		defer span.End()

		c, second := tracer.Start(c, "second_layer")
		time.Sleep(time.Second)
		c, third1 := tracer.Start(c, "third_layer_1")
		time.Sleep(100 * time.Millisecond)
		third1.End()
		c, third2 := tracer.Start(c, "third_layer_2")
		time.Sleep(300 * time.Millisecond)
		third2.End()
		second.End()
		ctx.RespStatusCode = 200
		ctx.RespData = []byte("hello, world")
	})

	s.Use((&MiddlewareBuilder{Tracer: tracer}).Build())
	s.Start(":8081")
}
