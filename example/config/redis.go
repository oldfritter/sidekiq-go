package config

import (
	"time"

	"github.com/go-redis/redis"
)

var (
	RedisCluster struct {
		Addrs       []string `yaml:"servers"`
		Password    string   `yaml:"password"`
		Dialtimeout int64    `yaml:"timeout"`
		Poolsize    int      `yaml:"pool"`
	}
	clusterClient *redis.ClusterClient
	client        *redis.Client
)

//
func ClusterClient() *redis.ClusterClient {
	return clusterClient
}

func Client() *redis.Client {
	return client
}
func CloseClusterClient() error {
	return clusterClient.Close()
}

func InitRedisCluster() {
	clusterClient = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:       RedisCluster.Addrs,
		Password:    RedisCluster.Password,
		DialTimeout: time.Second * time.Duration(RedisCluster.Dialtimeout),
		PoolSize:    RedisCluster.Poolsize,
	})
	_, err := clusterClient.Ping().Result()
	if err != nil {
		panic("redis connect error")
	}

	client = redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:6379",
		Password:    RedisCluster.Password,
		DialTimeout: time.Second * time.Duration(RedisCluster.Dialtimeout),
		PoolSize:    RedisCluster.Poolsize,
	})
	_, err = client.Ping().Result()
	if err != nil {
		panic("redis connect error")
	}
}
