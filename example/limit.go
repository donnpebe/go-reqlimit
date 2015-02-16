package main

import (
	"fmt"
	"html"
	"log"
	"net/http"

	"github.com/donnpebe/go-reqlimit"
)

const (
	namespace = "Appname"
	redishost = "localhost:6379"
	redispass = "cr4b"
	redispool = 25
)

var rpm *reqlimit.ReqLimiter

func requestLimiter(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exc, err := rpm.Exceed(r)
		if err != nil {
			log.Printf("Error: %v\n", err)
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if exc {
			http.Error(w, "Request limit exceeded", http.StatusForbidden)
			return
		}
		h(w, r)
	}
}

func main() {
	rq := reqlimit.New(namespace, redishost, redispass, redispool)
	defer rq.Close()

	rpm = rq.NewLimiter("rps", 60, 20)
	http.HandleFunc("/limit", requestLimiter(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello and welcome to %q", html.EscapeString(r.URL.Path))
	}))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
