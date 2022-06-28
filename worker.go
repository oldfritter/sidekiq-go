package sidekiq

import (
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
	Execute(*Exception) error
}

func Run(worker WorkerI) (idle bool, err error) {
	worker.Start()
	redisConn := worker.GetRedisConn()
	r, e := redisConn.Do("LPOP", worker.GetQueue())
	if r == nil || e != nil {
		idle = true
		return
	}
	v, e := redis.String(r, e)
	if v == "" {
		idle = true
	}
	worker.SetPayload(v)
	worker.Processing()
	if err == Stoping {
		worker.Fail()
		worker.LogError(v, err)
	} else {
		exception := Exception{}
		if err = worker.Execute(&exception); err != nil || exception.Msg != "" {
			worker.Fail()
			worker.LogError(v, err)
		}
	}
	worker.Processed()
	if err == Stoping {
		worker.LogInfo(" waiting to exit ......")
		time.Sleep(time.Second * 100)
	}
	return
}
