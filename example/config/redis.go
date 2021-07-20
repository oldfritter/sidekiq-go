package config

import (
	"log"
	"time"

	"github.com/gomodule/redigo/redis"
)

var (
	RedisConfig struct {
		Address     string `yaml:"server"`
		DB          int    `yaml:"db"`
		Password    string `yaml:"password"`
		IdleTimout  string `yaml:"timeout"`
		Poolsize    int    `yaml:"pool"`
		MaxCapacity int    `yaml:"maxopen"`
	}
	Pool *redis.Pool
)

func InitRedis() {
	timeOut, _ := time.ParseDuration(RedisConfig.IdleTimout)
	Pool = &redis.Pool{
		MaxIdle:     RedisConfig.Poolsize,
		MaxActive:   RedisConfig.MaxCapacity,
		IdleTimeout: timeOut,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", RedisConfig.Address)
			if err != nil {
				log.Println("redis can't dial:" + err.Error())
				return nil, err
			}

			if RedisConfig.Password != "" {
				_, err := conn.Do("AUTH", RedisConfig.Password)
				if err != nil {
					log.Println("redis can't AUTH:" + err.Error())
					conn.Close()
					return nil, err
				}
			}

			if RedisConfig.DB != 0 {
				_, err := conn.Do("SELECT", RedisConfig.DB)
				if err != nil {
					log.Println("redis can't SELECT:" + err.Error())
					conn.Close()
					return nil, err
				}
			}
			return conn, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			if err != nil {
				log.Println("redis can't ping, err:" + err.Error())
			}
			return err
		},
	}
}

func CloseRedisPool() {
	Pool.Close()
}

func GetRedisConn() redis.Conn {
	return Pool.Get()
}
