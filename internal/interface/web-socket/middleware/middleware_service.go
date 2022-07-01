package middleware

import "net/http"

type Service interface {
	WrapHandlerWithMiddlewares(
		handlerFunc http.HandlerFunc,
		middlewares ...Middleware,
	) http.HandlerFunc
}

type middlewareService struct {
}

type Middleware func(handlerFunc http.HandlerFunc) http.HandlerFunc

func NewMiddlewareService() Service {
	return &middlewareService{}
}

func (m *middlewareService) WrapHandlerWithMiddlewares(
	handlerFunc http.HandlerFunc,
	middlewares ...Middleware,
) http.HandlerFunc {
	if len(middlewares) < 1 {
		return handlerFunc
	}

	wrappedHandler := handlerFunc
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrappedHandler = middlewares[i](wrappedHandler)
	}

	return wrappedHandler
}
