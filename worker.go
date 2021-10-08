package sidekiq

import (
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
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

	GetName() string

	GetQueue() string
	GetQuerySize() int
	GetQueueErrors() string
	GetMaxQuery() int

	GetLog() string
	GetLogFolder() string
	LogInfo(text ...interface{})
	LogDebug(text ...interface{})
	LogError(text ...interface{})

	SetPayload(string)
	SetConn(redis.Conn)
	GetConn() redis.Conn
	GetRedisConn() redis.Conn

	Processing()
	Processed()
	Work() error
	Fail()
	Success()
	ReRunErrors()

	Perform(map[string]string)
	Priority(map[string]string)

	IsReady() bool
	Start()
	Stop()
	Recycle()
}

func Run(worker WorkerI) (idle bool, err error) {
	worker.Start()
	redisConn := worker.GetRedisConn()
	vs, _ := redis.Strings(redisConn.Do("LPOP", worker.GetQueue(), 1))
	if len(vs) < 1 {
		idle = true
	}
	for _, v := range vs {
		worker.SetPayload(v)
		worker.Processing()
		if err == Stoping {
			worker.Fail()
			worker.LogError(v, err)
		} else {
			exception := Exception{}
			if err = execute(worker, &exception); err != nil || exception.Msg != "" {
				worker.Fail()
				worker.LogError(v, err)
			}
		}
		worker.Processed()
	}
	if err == Stoping {
		worker.LogInfo(" waiting to exit ......")
		time.Sleep(time.Second * 100)
	}
	return
}

func execute(worker WorkerI, exception *Exception) (err error) {
	defer func(e *Exception) {
		r := recover()
		if r != nil {
			err = r.(error)
			e.Msg = fmt.Sprintf("%v", r)
		}
	}(exception)
	if worker.IsReady() {
		if err = worker.Work(); err == nil {
			worker.Success()
		}
	} else {
		err = Stoping
		worker.LogInfo(" skip for exit ......")
	}
	return
}
