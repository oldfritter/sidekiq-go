package sidekiq

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis"
)

const (
	DefaultLog   = "logs/workers.log"
	DefaultQueue = "sidekiq-go:queue:default"
)

type Exception struct {
	Msg string
}

type WorkerI interface {
	GetRedisClient() RedisClient
	GetName() string
	GetQueue() string
	RegisterQueue()
	GetLog() string
	GetLogFolder() string
	GetThreads() int
	SetPayload(*Payload)
	InitLogger()
	Lock(int)
	Unlock(int)
	IsLocked(int) bool
	Processing()
	Processed()
	SetClusterClient(*redis.ClusterClient)
	SetClient(*redis.Client)
	Work() error
	Fail()
	Success()
	ReRunErrors()
	FailProcessing()
}

type RedisClient interface {
	Do(...interface{}) *redis.Cmd
}

type Payload struct {
	Id        int       `json:"id"`
	Retry     int       `json:"retry"`
	DelayedAt time.Time `json:"delayed_at"`
	Notice    string    `json:"notic"`
}

func Run(worker WorkerI) (idle bool, err error) {
	redisClient := worker.GetRedisClient()
	cmd := redisClient.Do("BRPOP", worker.GetQueue(), 1)
	if cmd.Val() == nil {
		idle = true
		return
	}
	vs := cmd.Val().([]interface{})
	var t Payload
	if err = json.Unmarshal([]byte(vs[1].(string)), &t); err != nil {
		worker.Fail()
	} else {
		worker.SetPayload(&t)
		if locked := worker.IsLocked(t.Id); locked {
			return
		}
		worker.Lock(t.Id)
		worker.Processing()
		defer func() {
			worker.Unlock(t.Id)
			worker.Processed()
		}()
		exception := Exception{}
		if e := excute(worker, &exception); e != nil || exception.Msg != "" {
			worker.Fail()
		}
	}
	return
}

func excute(worker WorkerI, exception *Exception) (err error) {
	defer func(e *Exception) {
		r := recover()
		if r != nil {
			e.Msg = fmt.Sprintf("%v", r)
		}
	}(exception)
	err = worker.Work()
	// panic时，将不走以下代码
	if err != nil {
		worker.Fail()
	} else {
		worker.Success()
	}
	return
}
