package redis

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	redisPool *redis.Pool
)

// GetConnection returns a Redis connection from connection pool or error if no
// connection is available. Remember to use Close() after the connection is no
// longer needed to avoid memory leaks and too many Redis connection in use.
func GetConnection() (redis.Conn, error) {
	log.Trace().
		Str("method", "GetConnection").
		Msg("Borrow Redis connection from pool")

	if redisPool == nil {
		return nil, errors.New("Redis pool not initialized")
	}

	conn := redisPool.Get()

	return conn, nil
}

// InitPool initializes a Redis connection pool with max. 10 idle connections,
// 100 max active connections and 240 sec timeout.
func InitPool(host string, port int) {

	var address strings.Builder
	address.WriteString(host)
	address.WriteString(":")
	address.WriteString(strconv.Itoa(port))

	log.Debug().
		Str("method", "InitPool").
		Msgf("Initializing Redis pool")

	redisPool = &redis.Pool{
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		MaxActive:   100, // max number of connections
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", address.String())
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	cleanupHook()
}

// cleanupHook cleans up resources and shuts down gracefully.
func cleanupHook() {
	log.Trace().
		Str("method", "cleanupHook").
		Msg("Waiting for signal to exit gracefully")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGKILL)
	go func() {
		<-c
		redisPool.Close()
		os.Exit(0)
	}()
}

// Publish takes a Redis connection, the channel name and the message which
// should be published to that channel.
func Publish(conn redis.Conn, channel string, message []byte) error {
	log.Debug().
		Str("method", "Publish").
		Msgf("Publish to channel %s: %s", channel, string(message))

	_, err := conn.Do("PUBLISH", channel, message)
	if err != nil {
		v := string(message)
		if len(v) > 15 {
			v = v[0:12] + "..."
		}
		return fmt.Errorf("Error publish %s to %s: %v", v, channel, err)
	}

	return nil
}

// ExistsBoard checks if a board already exists in Redis database
func ExistsBoard(conn redis.Conn, key string) (bool, error) {
	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return false, err
	}

	return exists, nil
}

// CountPictures returns the number of pictures a board has
func CountPictures(conn redis.Conn, key string) (int, error) {
	count, err := redis.Int(conn.Do("SCARD", key))
	if err != nil {
		return 0, err
	}

	return count, nil
}

// AddPicture adds a picture to a key
func AddPicture(conn redis.Conn, key string, picture string) error {
	_, err := conn.Do("SADD", key, picture)
	if err != nil {
		return err
	}

	return nil
}
