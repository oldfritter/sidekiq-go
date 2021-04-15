package sidekiq

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/go-redis/redis"
)

type Worker struct {
	Name             string `yaml:"name"`
	Queue            string `yaml:"queue"`
	Log              string `yaml:"log"`
	Threads          int    `yaml:"threads"`
	UseDefaultPrefix bool   `yaml:"use_default_prefix"`
	Payload          *Payload
	Client           *redis.Client
	ClusterClient    *redis.ClusterClient
	logger           *log.Logger
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
	if worker.UseDefaultPrefix {
		return "sidekiq-go:" + worker.Queue
	}
	return worker.Queue
}

func (worker *Worker) GetQueueProcessing() string {
	return worker.Queue + ":processing"
}

func (worker *Worker) GetQueueErrors() string {
	return worker.Queue + ":errors"
}

func (worker *Worker) GetQueueDelay() string {
	return worker.Queue + ":delay"
}

func (worker *Worker) GetQueueDone() string {
	return worker.Queue + ":done"
}

func (worker *Worker) GetQueueFailed() string {
	return worker.Queue + ":failed"
}

func (worker *Worker) RegisterQueue() {
	cmd := worker.GetRedisClient().Do("LPOS", DefaultQueue, worker.GetQueue())
	if cmd.Val() == nil {
		worker.GetRedisClient().Do("RPUSH", DefaultQueue, worker.GetQueue())
	} else {
		worker.GetRedisClient().Do("LSET", DefaultQueue, cmd.Val(), worker.GetQueue())
	}
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

func (worker *Worker) GetThreads() int {
	return worker.Threads
}

func (worker *Worker) SetPayload(payload *Payload) {
	worker.Payload = payload
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

func (worker *Worker) Work() (err error) {
	return err
}

func (worker *Worker) Lock(id int) {
	worker.GetRedisClient().Do("SETEX", fmt.Sprintf("%v:lock:%v", worker.GetQueue(), id), 60, true)
}

func (worker *Worker) Unlock(id int) {
	worker.GetRedisClient().Do("DEL", fmt.Sprintf("%v:lock:%v", worker.GetQueue(), id))
}

func (worker *Worker) IsLocked(id int) (locked bool) {
	cmd := worker.GetRedisClient().Do("GET", fmt.Sprintf("%v:lock:%v", worker.GetQueue(), id))
	if cmd.Val() != nil {
		locked = true
	}
	return
}

func (worker *Worker) Processing(msg string) {
	worker.GetRedisClient().Do("SADD", worker.GetQueueProcessing(), msg)
}

func (worker *Worker) Processed(msg string) {
	worker.GetRedisClient().Do("SREM", worker.GetQueueProcessing(), msg)
}

func (worker *Worker) Fail(errMsg string) {
	client := worker.GetRedisClient()
	client.Do("SADD", worker.GetQueueErrors(), errMsg)
	client.Do("INCR", worker.GetQueueFailed())
	worker.LogError(errMsg)
}

func (worker *Worker) Success() {
	client := worker.GetRedisClient()
	client.Do("INCR", worker.GetQueueDone())
}

func (worker *Worker) SetClusterClient(client *redis.ClusterClient) {
	worker.ClusterClient = client
}

func (worker *Worker) SetClient(client *redis.Client) {
	worker.Client = client
}

func (worker *Worker) ReRunErrors(msg string) {
	worker.GetRedisClient().Do("SMOVE", worker.GetQueueErrors(), worker.GetQueue(), msg)
}

func (worker *Worker) FailProcessing(msg string) {
	worker.GetRedisClient().Do("SMOVE", worker.GetQueueProcessing(), worker.GetQueueErrors(), msg)
}
