package sidekiq

import (
	"encoding/json"
	"fmt"
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
	SetPayload(*Payload)
	InitLogger()
	Lock(int)
	Unlock(int)
	IsLocked(int) bool
	SetClusterClient(*redis.ClusterClient)
	SetClient(*redis.Client)
	Work() error
	Fail(string)
	Success()
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
	cmd := redisClient.Do("SPOP", worker.GetQueue(), 10)
	if cmd.Val() == nil {
		idle = true
		return
	}
	vs := cmd.Val().([]interface{})
	if len(vs) < 1 {
		idle = true
		return
	} else if len(vs) < 10 {
		idle = true
	}
	for _, v := range vs {
		var t Payload
		if locked := worker.IsLocked(t.Id); locked {
			continue
		}
		worker.Lock(t.Id)
		defer worker.Unlock(t.Id)
		if err := json.Unmarshal([]byte(v.(string)), &t); err != nil {
			worker.Fail(v.(string))
		} else {
			worker.SetPayload(&t)
			exception := Exception{}
			if e := excute(worker, &exception); e != nil || exception.Msg != "" {
				worker.Fail(v.(string))
			}
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
		worker.Fail(err.Error())
	} else {
		worker.Success()
	}
	return
}
