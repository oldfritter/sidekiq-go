package sidekiq

import (
	"log"
	"time"

	"github.com/go-redis/redis"
)

const (
	DefaultLog = "logs/workers.log"
)

type Exception struct {
	Msg string
}

type WorkerI interface {
	GetRedisClient() RedisClient
	GetName() string
	GetQueue() string
	GetLog() string
	GetLogFolder() string
	GetThreads() int
	GetPayload() *Payload
	SetPayload(*Payload)
	InitLogger()
	Lock(int)
	Unlock(int)
	SetClusterClient(*redis.ClusterClient)
	SetClient(*redis.Client)
	Work() error
	Fail(string)
}

type RedisClient interface {
	Do(...interface{}) *redis.Cmd
}

type Worker struct {
	Name          string `yaml:"name"`
	Queue         string `yaml:"queue"`
	Log           string `yaml:"log"`
	Threads       int    `yaml:"threads"`
	Payload       *Payload
	Client        *redis.Client
	ClusterClient *redis.ClusterClient
	logger        *log.Logger
}

type Payload struct {
	Id        int       `json:"id"`
	Retry     int       `json:"retry"`
	DelayedAt time.Time `json:"delayed_at"`
	Notice    string    `json:"notic"`
}
