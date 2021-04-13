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

func (worker *Worker) GetQueueError() string {
	return worker.Queue + ":error"
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

func (worker *Worker) GetPayload() *Payload {
	return worker.Payload
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
	fmt.Println("worker err: ", err)
	return err
}

func (worker *Worker) Lock(id int) {
	client := worker.GetRedisClient()
	key := fmt.Sprintf("%v:lock:%v", worker.GetQueue(), id)
	client.Do("SETEX", key, 60, true)
}

func (worker *Worker) Unlock(id int) {
	client := worker.GetRedisClient()
	key := fmt.Sprintf("%v:lock:%v", worker.GetQueue(), id)
	client.Do("Del", key)
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
		if err := json.Unmarshal([]byte(v.(string)), &t); err != nil {
			worker.Fail(v.(string))
		} else {
			fmt.Println("worker:", worker)
			fmt.Println("worker name:", worker.GetName())
			worker.SetPayload(&t)
			if err := worker.Work(); err != nil {
				worker.Fail(v.(string))
			}
		}
	}
	return
}

func (worker *Worker) Fail(errMsg string) {
	client := worker.GetRedisClient()
	client.Do("SADD", worker.GetQueueErrors(), errMsg)
	client.Do("INCR", worker.GetQueueError())
	worker.LogError(errMsg)
}

func (worker *Worker) SetClusterClient(client *redis.ClusterClient) {
	worker.ClusterClient = client
}

func (worker *Worker) SetClient(client *redis.Client) {
	worker.Client = client
}
