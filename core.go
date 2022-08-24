package sidekiq

import (
	"encoding/json"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/gomodule/redigo/redis"
)

type Worker struct {
	Name     string `yaml:"name"`
	Queue    string `yaml:"queue"`
	Log      string `yaml:"log"`
	MaxQuery int    `yaml:"maxQuery"`
	Threads  int    `yaml:"threads"`
	Prefix   string `yaml:"prefix"`
	Payload  string
	Ready    bool
	Conn     redis.Conn
	logger   *log.Logger
}

func (worker *Worker) InitLogger() {
	err := os.Mkdir(worker.GetLogFolder(), 0755)
	if err != nil {
		if !os.IsExist(err) {
			log.Fatalf("create folder error: %v", err)
		}
	}
	file, err := os.OpenFile(worker.GetLog(), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("open file error: %v", err)
	}
	worker.logger = log.New(file, "", log.LstdFlags)
}

func (worker *Worker) RegisterQueue() {
	conn := worker.GetRedisConn()
	cmd, _ := redis.String(conn.Do("LREM", DefaultQueue, 0, worker.GetQueue()))
	if cmd == "" {
		conn.Do("RPUSH", DefaultQueue, worker.GetQueue())
	} else {
		conn.Do("LSET", DefaultQueue, cmd, worker.GetQueue())
	}
}

func (worker *Worker) GetRedisConn() redis.Conn {
	return worker.Conn
}

func (worker *Worker) GetName() string {
	return worker.Name
}

func (worker *Worker) GetQueue() string {
	if worker.Prefix != "" {
		return worker.Prefix + ":" + worker.Queue
	}
	return worker.Queue
}

func (worker *Worker) GetQuerySize() int {
	conn := worker.GetRedisConn()
	l, _ := redis.Int(conn.Do("LLEN", worker.GetQueue()))
	return l
}

func (worker *Worker) GetQueueProcessing() string {
	return worker.GetQueue() + ":processing"
}

func (worker *Worker) GetQueueErrors() string {
	return worker.GetQueue() + ":errors"
}

func (worker *Worker) GetQueueDelay() string {
	return worker.GetQueue() + ":delay"
}

func (worker *Worker) GetQueueDone() string {
	return worker.GetQueue() + ":done"
}

func (worker *Worker) GetQueueFailed() string {
	return worker.GetQueue() + ":failed"
}

func (worker *Worker) GetLog() string {
	if worker.Log != "" {
		return worker.Log
	}
	return DefaultLog
}

func (worker *Worker) GetLogFolder() string {
	var re = regexp.MustCompile(`/.*\.log$`)
	return strings.TrimSuffix(worker.GetLog(), re.FindString(worker.GetLog()))
}

func (worker *Worker) GetMaxQuery() int {
	return worker.MaxQuery
}

func (worker *Worker) SetPayload(payload string) {
	worker.Payload = payload
}

func (worker *Worker) SetConn(conn redis.Conn) {
	worker.Conn = conn
}

func (worker *Worker) GetConn() redis.Conn {
	return worker.Conn
}

func (worker *Worker) Work() (err error) {
	return
}

func (worker *Worker) Processing() {
	worker.GetRedisConn().Do("LREM", worker.GetQueueProcessing(), 0, worker.Payload)
	worker.GetRedisConn().Do("RPUSH", worker.GetQueueProcessing(), worker.Payload)
	worker.GetRedisConn().Do("LREM", worker.GetQueue(), 0, worker.Payload)
}

func (worker *Worker) Processed() {
	worker.GetRedisConn().Do("LREM", worker.GetQueueProcessing(), 0, worker.Payload)
}

func (worker *Worker) Fail() {
	conn := worker.GetRedisConn()
	conn.Do("LREM", worker.GetQueueErrors(), 0, worker.Payload)
	conn.Do("RPUSH", worker.GetQueueErrors(), worker.Payload)
	conn.Do("INCR", worker.GetQueueFailed())
}

func (worker *Worker) Success() {
	conn := worker.GetRedisConn()
	conn.Do("INCR", worker.GetQueueDone())
}

func (worker *Worker) ReRunErrors() {
	conn := worker.GetRedisConn()
	size, _ := redis.Int(conn.Do("LLEN", worker.GetQueueErrors()))
	for i := 0; i < size; i++ {
		conn.Do("LMOVE", worker.GetQueueErrors(), worker.GetQueue(), "LEFT", "RIGHT")
	}
}

func (worker *Worker) Perform(message map[string]string) {
	var lock sync.Mutex
	func() {
		lock.Lock()
		defer lock.Unlock()
		b, _ := json.Marshal(message)
		worker.GetRedisConn().Do("LREM", worker.GetQueue(), 0, string(b))
		worker.GetRedisConn().Do("RPUSH", worker.GetQueue(), string(b))
	}()
}

func (worker *Worker) Priority(message map[string]string) {
	var lock sync.Mutex
	func() {
		lock.Lock()
		defer lock.Unlock()
		b, _ := json.Marshal(message)
		worker.GetRedisConn().Do("LREM", worker.GetQueue(), 0, string(b))
		worker.GetRedisConn().Do("LPUSH", worker.GetQueue(), string(b))
	}()
}

func (worker *Worker) IsReady() bool {
	return worker.Ready
}

func (worker *Worker) Start() {
	worker.Ready = true
}

func (worker *Worker) Stop() {
	worker.Ready = false
}

func (worker *Worker) Recycle() {
	conn := worker.GetRedisConn()
	size, _ := redis.Int(conn.Do("LLEN", worker.GetQueueProcessing()))
	for i := 0; i < size; i++ {
		conn.Do("LMOVE", worker.GetQueueProcessing(), worker.GetQueue(), "LEFT", "RIGHT")
	}
}

func (worker *Worker) Execute() {
	return
}
