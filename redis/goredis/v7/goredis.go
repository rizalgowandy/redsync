package goredis

import (
	"context"
	"strings"
	"time"

	"github.com/go-redis/redis/v7"
	redsyncredis "github.com/go-redsync/redsync/v4/redis"
)

type pool struct {
	delegate redis.UniversalClient
}

func (p *pool) Get(ctx context.Context) (redsyncredis.Conn, error) {
	c := p.delegate
	if ctx != nil {
		switch client := p.delegate.(type) {
		case *redis.Client:
			c = client.WithContext(ctx)
		case *redis.ClusterClient:
			c = client.WithContext(ctx)
		}
	}
	return &conn{c}, nil
}

// NewPool returns a Goredis-based pool implementation.
func NewPool(delegate redis.UniversalClient) redsyncredis.Pool {
	return &pool{delegate}
}

type conn struct {
	delegate redis.UniversalClient
}

func (c *conn) Get(name string) (string, error) {
	value, err := c.delegate.Get(name).Result()
	return value, noErrNil(err)
}

func (c *conn) Set(name string, value string) (bool, error) {
	reply, err := c.delegate.Set(name, value, 0).Result()
	return reply == "OK", err
}

func (c *conn) SetNX(name string, value string, expiry time.Duration) (bool, error) {
	return c.delegate.SetNX(name, value, expiry).Result()
}

func (c *conn) PTTL(name string) (time.Duration, error) {
	return c.delegate.PTTL(name).Result()
}

func (c *conn) Eval(script *redsyncredis.Script, keysAndArgs ...interface{}) (interface{}, error) {
	keys := make([]string, script.KeyCount)
	args := keysAndArgs

	if script.KeyCount > 0 {
		for i := range script.KeyCount {
			keys[i] = keysAndArgs[i].(string)
		}
		args = keysAndArgs[script.KeyCount:]
	}

	v, err := c.delegate.EvalSha(script.Hash, keys, args...).Result()
	if err != nil && strings.HasPrefix(err.Error(), "NOSCRIPT ") {
		v, err = c.delegate.Eval(script.Src, keys, args...).Result()
	}
	return v, noErrNil(err)
}

func (c *conn) Close() error {
	// Not needed for this library
	return nil
}

func noErrNil(err error) error {
	if err == redis.Nil {
		return nil
	}
	return err
}
