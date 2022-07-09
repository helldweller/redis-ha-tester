package app

import (
	"github.com/go-redis/redis/v8"
	// "fmt"
)

var rdb *redis.ClusterClient
var redisNil = redis.Nil

func init() {
	rdb = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{conf.RedisClusterHost},
		// To route commands by latency or randomly, enable one of the following.
		//RouteByLatency: true,
		//RouteRandomly: true,
	})
}
