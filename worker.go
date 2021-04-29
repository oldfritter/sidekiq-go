package sidekiq

import (
	"encoding/json"
	"fmt"
	"log"
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
	InitLogger()
	RegisterQueue()

	GetRedisClient() RedisClient
	GetName() string
	GetQueue() string
	GetQuerySize() int64
	GetQueueErrors() string
	GetLog() string
	GetLogFolder() string
	GetMaxQuery() int64

	SetPayload(Payload)
	SetClusterClient(*redis.ClusterClient)
	SetClient(*redis.Client)

	Lock(string)
	Unlock(string)
	Processing()
	Processed()
	Work() error
	Fail()
	Success()
	ReRunErrors()
	FailProcessing()
	Perform(map[string]string)
	Priority(map[string]string)

	IsLocked(string) bool
	IsReady() bool
	Start()
	Stop()
	Recycle()
}

type RedisClient interface {
	Do(...interface{}) *redis.Cmd
}

type Payload map[string]string

// 阻塞：按照进入队列的顺序执行
func SortedRun(worker WorkerI) (idle bool, err error) {
	worker.Start()
	redisClient := worker.GetRedisClient()
	cmd := redisClient.Do("BRPOP", worker.GetQueue(), 1)
	if cmd.Val() == nil {
		idle = true
		return
	}
	vs := cmd.Val().([]interface{})
	var t Payload
	if err = json.Unmarshal([]byte(vs[1].(string)), t); err != nil {
		worker.Fail()
	} else {
		worker.SetPayload(t)
		if locked := worker.IsLocked(t["id"]); locked {
			return
		}
		worker.Lock(t["id"])
		worker.Processing()
		exception := Exception{}
		if e := excute(worker, &exception); e != nil || exception.Msg != "" {
			worker.Fail()
		} else {
			worker.Processed()
		}
		worker.Unlock(t["id"])
	}
	return
}

// 非阻塞：无需按顺序执行
func Run(worker WorkerI) (idle bool, err error) {
	worker.Start()
	redisClient := worker.GetRedisClient()
	cmd := redisClient.Do("RPOP", worker.GetQueue(), 3)
	if cmd.Val() == nil {
		idle = true
		return
	}
	vs := cmd.Val().([]interface{})
	if len(vs) < 3 {
		idle = true
	}
	for _, v := range vs {
		var t Payload
		if err = json.Unmarshal([]byte(v.(string)), &t); err != nil {
			worker.Fail()
		} else {
			worker.SetPayload(t)
			if locked := worker.IsLocked(t["id"]); locked {
				continue
			}
			worker.Lock(t["id"])
			worker.Processing()
			exception := Exception{}
			if e := excute(worker, &exception); e != nil || exception.Msg != "" {
				worker.Fail()
			} else {
				worker.Processed()
			}
			worker.Unlock(t["id"])
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
	if worker.IsReady() {
		err = worker.Work()
	} else {
		log.Println(worker.GetName(), " waiting to exit ......")
		time.Sleep(time.Second * 100)
	}
	// panic时，将不走以下代码
	if err != nil {
		// worker.Fail()
	} else {
		worker.Success()
	}
	return
}
