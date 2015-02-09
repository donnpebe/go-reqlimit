package reqlimit

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/garyburd/redigo/redis"
)

// ReqConfig control the reqlimit general configuration
type ReqConfig struct {
	namespace string
	redisPool *redis.Pool
	names     map[string]bool
}

// New create a new ReqLimiter
// Use it like this:
// New(namespace, redisHost, redisPass, redisPoolCount)
// namespace: namespace for redis key (optional, default empty)
// redisHost: redis host and port (optional, default localhost:6379)
// redisPass: redis password (optional, default empty)
// redisPoolCount: number of pool for redis connection (optional, default 5)
func New(attrs ...interface{}) *ReqConfig {
	cf := assignConfig(attrs)
	rc := &ReqConfig{
		namespace: cf.namespace,
		redisPool: redisPool(cf.redisHost, cf.redisPass, cf.poolCount),
		names:     make(map[string]bool),
	}
	return rc
}

// NewLimiter create request limiter for limit/interval (ex 20/second)
// interval must be in second
// you can make more than one limiter but name of limiter must be unique
// if there is duplicate in names, it will panic
func (rc *ReqConfig) NewLimiter(name string, interval int, limit int) *ReqLimiter {
	if rc.names[name] {
		panic("Name must be unique")
	}
	rc.names[name] = true

	rl := &ReqLimiter{
		name:      name,
		reqConfig: rc,
		interval:  interval,
		limit:     limit,
	}
	return rl
}

// Close close the connection use by reqlimiter, you must call this close
// after you done using req limiter
func (rc *ReqConfig) Close() {
	rc.redisPool.Close()
}

type config struct {
	namespace string
	redisHost string
	redisPass string
	poolCount int
}

func assignConfig(args []interface{}) *config {
	cf := &config{
		namespace: "",
		redisHost: "localhost:6379",
		redisPass: "",
		poolCount: 5,
	}
	if len(args) > 0 {
		cf.namespace = args[0].(string)
	}
	if len(args) > 1 {
		cf.redisHost = args[1].(string)
	}
	if len(args) > 2 {
		cf.redisPass = args[2].(string)
	}
	if len(args) > 3 {
		c := args[3].(int)
		if c > 0 {
			cf.poolCount = c
		}
	}
	return cf
}

// redisPool get a pool of redis connections
func redisPool(host, password string, count int) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     count,
		IdleTimeout: 25 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", host)
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

// ReqLimiter limit how many request can be made per interval Duration
type ReqLimiter struct {
	// limiter name
	name string
	// config
	reqConfig *ReqConfig
	// limiter duration in second
	interval int
	// limiter count
	limit int
}

// Exceed return true if request exceeded the limit, false otherwise
func (rl *ReqLimiter) Exceed(r *http.Request) (bool, error) {
	ip := realIPAddress(r)
	key := rl.limitKey(ip)
	num, err := rl.incr(key)
	if err != nil {
		return false, err
	}
	return num > rl.limit, nil
}

// limitKey redis key for limiter
func (rl *ReqLimiter) limitKey(ip string) string {
	if rl.reqConfig.namespace == "" {
		return fmt.Sprintf("limiter:%s:%s", rl.name, ip)
	}
	return fmt.Sprintf("%s:limiter:%s:%s", rl.reqConfig.namespace, rl.name, ip)
}

// incr the value of key
func (rl *ReqLimiter) incr(key string) (int, error) {
	conn := rl.reqConfig.redisPool.Get()
	defer conn.Close()

	n, err := redis.Int(conn.Do("INCR", key))
	if n == 1 {
		conn.Do("EXPIRE", key, rl.interval)
	}

	return n, err
}

// realIPAddress get the real ip behind proxy
func realIPAddress(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); len(ip) > 0 {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); len(ip) > 0 {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded"); len(ip) > 0 {
		return ip
	}
	if ip := r.Header.Get("Client-IP"); len(ip) > 0 {
		return ip
	}
	realip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return realip
}
