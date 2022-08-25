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

	GetLog() string
	GetLogFolder() string
	LogInfo(text ...interface{})
	LogDebug(text ...interface{})
	LogError(text ...interface{})

	SetPayload(string)
	SetRedisConn(redis.Conn)
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

	Execute() error
}

func Run(worker WorkerI) (idle, processed bool, err error) {
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
		return
	}
	worker.SetPayload(v)
	worker.Processing()
	if err == Stoping {
		worker.Fail()
		worker.LogError(v, err)
	} else {
		exception := Exception{}
		if err = Execute(worker, &exception); err != nil || exception.Msg != "" {
			worker.Fail()
			worker.LogError(v, err)
		} else {
			worker.Processed()
			processed = true
		}
	}
	if err == Stoping {
		worker.LogInfo(" waiting to exit ......")
		time.Sleep(time.Second * 10)
	}
	return
}

func Execute(worker WorkerI, exception *Exception) (err error) {
	defer func(e *Exception) {
		r := recover()
		if r != nil {
			err = r.(error)
			e.Msg = fmt.Sprintf("%v", r)
			worker.LogInfo(" recover for err: ", e.Msg)
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

func PureRun(worker WorkerI) (idle, processed bool, err error) {
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
		return
	}
	worker.SetPayload(v)
	worker.Processing()
	if err == Stoping {
		worker.Fail()
		worker.LogError(v, err)
	} else {
		if err = worker.Execute(); err != nil {
			worker.Fail()
			worker.LogError(v, err)
		} else {
			worker.Processed()
			processed = true
		}
	}
	if err == Stoping {
		worker.LogInfo(" waiting to exit ......")
		time.Sleep(time.Second * 10)
	}
	return
}
