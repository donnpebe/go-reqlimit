go-reqlimit
===========

**go-reqlimit** is a request limiter that limit the number of request per ip address. It use redis as storage system.

Usage
-----
- Get the code
```go get github.com/donnpebe/go-reqlimit```

- Import it
```Go
import "github.com/donnpebe/go-reqlimit"
```

- Use it in your code
Very basic use, create new ReqConfig and then create a new ReqLimiter
```Go
// It will assume that you have redis installed in localhost port 6379 and doesn't have a password 
rq := reqlimit.New()
// or if you want complete control of your redis connection
// rq := reqlimit.New("Namespace", "localhost:6379", "redispass", 25)
defer rq.Close()

// limiter with name "rps" and rate: 10req/second
rps := rq.NewLimiter("rps", 1, 10)
// you can have more than one limiter
// this limiter has name "rpm" and rate: 30req/minute
// name of limiter must be unique, or it will panic
rpm := rq.NewLimiter("rpm", 60, 30)

// test if request exceed the limit with Exceed method
// where r is *http.Request
if rps.Exceed(r) || rpm.Exceed(r) {
	http.Error(w, "Request limit exceeded", http.StatusForbidden)
}
```

Example
-------

```Go
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
```

License
-------

[MIT Public License](https://github.com/donnpebe/go-reqlimit/blob/master/LICENSE)
