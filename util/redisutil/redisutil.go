package redisutil

import (
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClientFromURL creates a new Redis client based on the provided URL.
// The URL scheme can be either `redis` or `redis+sentinel`.
func RedisClientFromURL(redisUrl string) (redis.UniversalClient, error) {
	if redisUrl == "" {
		return nil, nil
	}
	u, err := url.Parse(redisUrl)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "redis+sentinel" {
		redisOptions, err := parseFailoverRedisUrl(redisUrl)
		if err != nil {
			return nil, err
		}
		return redis.NewFailoverClient(redisOptions), nil
	}
	redisOptions, err := redis.ParseURL(redisUrl)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(redisOptions), nil
}

// Designed using https://github.com/redis/go-redis/blob/a8590e987945b7ba050569cc3b94b8ece49e99e3/options.go#L283 as reference
// Example Usage :
//
//	redis+sentinel://<user>:<password>@<host1>:<port1>,<host2>:<port2>,<host3>:<port3>/<master_name/><db_number>?dial_timeout=3&db=1&read_timeout=6s&max_retries=2
func parseFailoverRedisUrl(redisUrl string) (*redis.FailoverOptions, error) {
	u, err := url.Parse(redisUrl)
	if err != nil {
		return nil, err
	}
	o := &redis.FailoverOptions{}
	o.SentinelUsername, o.SentinelPassword = getUserPassword(u)
	o.SentinelAddrs = getAddressesWithDefaults(u)
	f := strings.FieldsFunc(u.Path, func(r rune) bool {
		return r == '/'
	})
	switch len(f) {
	case 0:
		return nil, fmt.Errorf("redis: master name is required")
	case 1:
		o.DB = 0
		o.MasterName = f[0]
	case 2:
		o.MasterName = f[0]
		var err error
		if o.DB, err = strconv.Atoi(f[1]); err != nil {
			return nil, fmt.Errorf("redis: invalid database number: %q", f[0])
		}
	default:
		return nil, fmt.Errorf("redis: invalid URL path: %s", u.Path)
	}

	return setupConnParams(u, o)
}

func getUserPassword(u *url.URL) (string, string) {
	var user, password string
	if u.User != nil {
		user = u.User.Username()
		if p, ok := u.User.Password(); ok {
			password = p
		}
	}
	return user, password
}

func getAddressesWithDefaults(u *url.URL) []string {
	urlHosts := strings.Split(u.Host, ",")
	var addresses []string
	for _, urlHost := range urlHosts {
		host, port, err := net.SplitHostPort(urlHost)
		if err != nil {
			host = u.Host
		}
		if host == "" {
			host = "localhost"
		}
		if port == "" {
			port = "6379"
		}
		addresses = append(addresses, net.JoinHostPort(host, port))
	}
	return addresses
}

type queryOptions struct {
	q   url.Values
	err error
}

func (o *queryOptions) has(name string) bool {
	return len(o.q[name]) > 0
}

func (o *queryOptions) string(name string) string {
	vs := o.q[name]
	if len(vs) == 0 {
		return ""
	}
	delete(o.q, name) // enable detection of unknown parameters
	return vs[len(vs)-1]
}

func (o *queryOptions) int(name string) int {
	s := o.string(name)
	if s == "" {
		return 0
	}
	i, err := strconv.Atoi(s)
	if err == nil {
		return i
	}
	if o.err == nil {
		o.err = fmt.Errorf("redis: invalid %s number: %w", name, err)
	}
	return 0
}

func (o *queryOptions) duration(name string) time.Duration {
	s := o.string(name)
	if s == "" {
		return 0
	}
	// try plain number first
	if i, err := strconv.Atoi(s); err == nil {
		if i <= 0 {
			// disable timeouts
			return -1
		}
		return time.Duration(i) * time.Second
	}
	dur, err := time.ParseDuration(s)
	if err == nil {
		return dur
	}
	if o.err == nil {
		o.err = fmt.Errorf("redis: invalid %s duration: %w", name, err)
	}
	return 0
}

func (o *queryOptions) bool(name string) bool {
	switch s := o.string(name); s {
	case "true", "1":
		return true
	case "false", "0", "":
		return false
	default:
		if o.err == nil {
			o.err = fmt.Errorf("redis: invalid %s boolean: expected true/false/1/0 or an empty string, got %q", name, s)
		}
		return false
	}
}

func (o *queryOptions) remaining() []string {
	if len(o.q) == 0 {
		return nil
	}
	keys := make([]string, 0, len(o.q))
	for k := range o.q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func setupConnParams(u *url.URL, o *redis.FailoverOptions) (*redis.FailoverOptions, error) {
	q := queryOptions{q: u.Query()}

	// compat: a future major release may use q.int("db")
	if tmp := q.string("db"); tmp != "" {
		db, err := strconv.Atoi(tmp)
		if err != nil {
			return nil, fmt.Errorf("redis: invalid database number: %w", err)
		}
		o.DB = db
	}

	o.Protocol = q.int("protocol")
	o.ClientName = q.string("client_name")
	o.MaxRetries = q.int("max_retries")
	o.MinRetryBackoff = q.duration("min_retry_backoff")
	o.MaxRetryBackoff = q.duration("max_retry_backoff")
	o.DialTimeout = q.duration("dial_timeout")
	o.ReadTimeout = q.duration("read_timeout")
	o.WriteTimeout = q.duration("write_timeout")
	o.PoolFIFO = q.bool("pool_fifo")
	o.PoolSize = q.int("pool_size")
	o.PoolTimeout = q.duration("pool_timeout")
	o.MinIdleConns = q.int("min_idle_conns")
	o.MaxIdleConns = q.int("max_idle_conns")
	o.MaxActiveConns = q.int("max_active_conns")
	if q.has("conn_max_idle_time") {
		o.ConnMaxIdleTime = q.duration("conn_max_idle_time")
	} else {
		o.ConnMaxIdleTime = q.duration("idle_timeout")
	}
	if q.has("conn_max_lifetime") {
		o.ConnMaxLifetime = q.duration("conn_max_lifetime")
	} else {
		o.ConnMaxLifetime = q.duration("max_conn_age")
	}
	if q.err != nil {
		return nil, q.err
	}

	// any parameters left?
	if r := q.remaining(); len(r) > 0 {
		return nil, fmt.Errorf("redis: unexpected option: %s", strings.Join(r, ", "))
	}

	return o, nil
}
