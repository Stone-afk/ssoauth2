package middleware

import "web/handler"

type Middleware func(next handler.HandleFunc) handler.HandleFunc
