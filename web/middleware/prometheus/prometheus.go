package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"time"
	"web/context"
	webHandler "web/handler"
	"web/middleware"
)

func (m *MiddlewareBuilder) Build() middleware.Middleware {
	//  创建一个 Vector （向量）其实就是观察者，设置 ConstLabels 和 Labels
	summaryVec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:        m.Name,
		Subsystem:   m.Subsystem,
		ConstLabels: m.ConstLabels,
		Help:        m.Help,
	}, []string{"pattern", "method", "status"})
	// 记得调用 MustRegister 把观 察者注册进去。
	prometheus.MustRegister(summaryVec)
	return func(next webHandler.HandleFunc) webHandler.HandleFunc {
		return func(ctx *context.Context) {
			startTime := time.Now()
			next(ctx)
			endTime := time.Now()
			go report(endTime.Sub(startTime), ctx, summaryVec)
		}
	}
}

func report(dur time.Duration, ctx *context.Context, vec prometheus.ObserverVec) {
	status := ctx.RespStatusCode
	route := "unknown"
	if ctx.MatchedRoute != "" {
		route = ctx.MatchedRoute
	}
	// 将响应时间记录成毫秒数
	ms := dur / time.Millisecond

	// 使用 WithLabelValues 来获得具体的收集器
	vec.WithLabelValues(
		route, ctx.Request.Method, strconv.Itoa(status)).Observe(float64(ms))
}

type MiddlewareBuilder struct {
	Name        string
	Subsystem   string
	ConstLabels map[string]string
	Help        string
}
