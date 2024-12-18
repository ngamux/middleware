package recover

import (
	"errors"
	"fmt"
	"log"
	"net/http"
)

var configDefault = Config{
	ErrorHandler: func(rw http.ResponseWriter, r *http.Request, e error) {
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(rw, e)
		log.Println("error:", e)
	},
}

func New(config ...Config) func(next http.HandlerFunc) http.HandlerFunc {
	cfg := configDefault

	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(rw http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					cfg.ErrorHandler(rw, r, errors.New(err.(string)))
				}
			}()
			next(rw, r)
		}
	}
}
