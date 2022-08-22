package middleware

import (
	dbpg "github.com/vulpemventures/neutrino-elements/internal/infrastructure/storage/db/pg"
	"net/http"
)

type Service interface {
	WrapHandlerWithMiddlewares(
		handlerFunc http.HandlerFunc,
		middlewares ...Middleware,
	) http.HandlerFunc
	LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc
	PanicRecovery(next http.HandlerFunc) http.HandlerFunc
}

type middlewareService struct {
	dbSvc dbpg.DbService
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
