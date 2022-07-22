package middleware

import (
	log "github.com/sirupsen/logrus"
	"net/http"
	"runtime/debug"
)

func (m *middlewareService) PanicRecovery(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				log.Println(string(debug.Stack()))
				log.Errorf("panic-recovery middleware recovered from panic: %v", err)
				log.Tracef("panic-recovery middleware recovered from panic: %v", string(debug.Stack()))
			}
		}()

		next(w, req)
	}
}
