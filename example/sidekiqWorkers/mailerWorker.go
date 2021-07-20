package sidekiqWorkers

import (
	"encoding/json"
	"fmt"
	"time"

	sidekiq "github.com/oldfritter/sidekiq-go/v1"
)

type Message struct {
	Action string `json:"action"`
	Method string `json:"method"`
	Email  string `json:"email"`
	Id     int    `json:"id"`
}

func CreateMailerWorker(w *sidekiq.Worker) sidekiq.WorkerI {
	return &MailerWorker{*w, Message{}}
}

type MailerWorker struct {
	sidekiq.Worker
	Payload Message // 自定义Payload的数据类型
}

// 重写数据解析
func (worker *MailerWorker) SetPayload(payload string) {
	json.Unmarshal([]byte(payload), &worker.Payload)
}

func (worker *MailerWorker) Work() (err error) {
	start := time.Now().UnixNano()
	fmt.Println("Payload: ", worker.Payload)
	// panic("test panic")

	// err = fmt.Errorf("test")

	// 这里完成此worker的功能

	worker.LogInfo("payload.Id: ", worker.Payload.Id, ", time:", (time.Now().UnixNano()-start)/1000000, " ms")
	return
}
