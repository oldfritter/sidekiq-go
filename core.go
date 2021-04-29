package sidekiq

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/go-redis/redis"
)

type Worker struct {
	Name          string `yaml:"name"`
	Queue         string `yaml:"queue"`
	Log           string `yaml:"log"`
	MaxQuery      int64  `yaml:"max_query"`
	Threads       int    `yaml:"threads"`
	DefaultPrefix bool   `yaml:"default_prefix"`
	Payload       Payload

	Client        *redis.Client
	ClusterClient *redis.ClusterClient
	logger        *log.Logger
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
	cmd := worker.GetRedisClient().Do("LPOS", DefaultQueue, worker.GetQueue())
	if cmd.Val() == nil {
		worker.GetRedisClient().Do("RPUSH", DefaultQueue, worker.GetQueue())
	} else {
		worker.GetRedisClient().Do("LSET", DefaultQueue, cmd.Val(), worker.GetQueue())
	}
}

func (worker *Worker) GetRedisClient() RedisClient {
	if worker.ClusterClient != nil {
		return worker.ClusterClient
	}
	return worker.Client
}

func (worker *Worker) GetName() string {
	return worker.Name
}

func (worker *Worker) GetQueue() string {
	if worker.DefaultPrefix {
		return "sidekiq-go:" + worker.Queue
	}
	return worker.Queue
}

func (worker *Worker) GetQuerySize() int64 {
	client := worker.GetRedisClient()
	return client.Do("LLEN", worker.GetQueue()).Val().(int64)
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

func (worker *Worker) GetMaxQuery() int64 {
	return worker.MaxQuery
}

func (worker *Worker) SetPayload(payload Payload) {
	worker.Payload = payload
}

func (worker *Worker) SetClusterClient(client *redis.ClusterClient) {
	worker.ClusterClient = client
}

func (worker *Worker) SetClient(client *redis.Client) {
	worker.Client = client
}

func (worker *Worker) Work() (err error) {
	return err
}

func (worker *Worker) Lock(id string) {
	worker.GetRedisClient().Do("SETEX", fmt.Sprintf("%v:lock:%v", worker.GetQueue(), id), 600, true)
}

func (worker *Worker) Unlock(id string) {
	worker.GetRedisClient().Do("DEL", fmt.Sprintf("%v:lock:%v", worker.GetQueue(), id))
}

func (worker *Worker) Processing() {
	b, _ := json.Marshal(worker.Payload)
	worker.GetRedisClient().Do("LPUSH", worker.GetQueueProcessing(), string(b))
}

func (worker *Worker) Processed() {
	b, _ := json.Marshal(worker.Payload)
	worker.GetRedisClient().Do("LREM", worker.GetQueueProcessing(), 0, string(b))
}

func (worker *Worker) Fail() {
	b, _ := json.Marshal(worker.Payload)
	client := worker.GetRedisClient()
	client.Do("LPUSH", worker.GetQueueErrors(), string(b))
	client.Do("INCR", worker.GetQueueFailed())
	worker.LogError(string(b))
}

func (worker *Worker) Success() {
	client := worker.GetRedisClient()
	client.Do("INCR", worker.GetQueueDone())
}

func (worker *Worker) ReRunErrors() {
	client := worker.GetRedisClient()
	size := client.Do("LLEN", worker.GetQueueErrors()).Val().(int64)
	for i := int64(0); i < size; i++ {
		client.Do("LMOVE", worker.GetQueueErrors(), worker.GetQueue(), "LEFT", "RIGHT")
	}
}

func (worker *Worker) FailProcessing() {
	b, _ := json.Marshal(worker.Payload)
	worker.GetRedisClient().Do("BLMOVE", worker.GetQueueProcessing(), worker.GetQueueErrors(), string(b))
}

func (worker *Worker) Perform(message map[string]string) {
	b, _ := json.Marshal(message)
	if worker.IsLocked(message["id"]) {
		return
	} else {
		worker.GetRedisClient().Do("LREM", worker.GetQueue(), 1000, string(b))
		worker.GetRedisClient().Do("LPUSH", worker.GetQueue(), string(b))
	}
}

func (worker *Worker) Priority(message map[string]string) {
	b, _ := json.Marshal(message)
	if worker.IsLocked(message["id"]) {
		return
	} else {
		worker.GetRedisClient().Do("LREM", worker.GetQueue(), 1000, string(b))
		worker.GetRedisClient().Do("RPUSH", worker.GetQueue(), string(b))
	}
}

func (worker *Worker) IsLocked(id string) (locked bool) {
	cmd := worker.GetRedisClient().Do("GET", fmt.Sprintf("%v:lock:%v", worker.GetQueue(), id))
	if cmd.Val() != nil {
		locked = true
	}
	return
}
