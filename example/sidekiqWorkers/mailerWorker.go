package sidekiqWorkers

import (
	"encoding/json"
	"fmt"
	"time"

	sidekiq "github.com/oldfritter/sidekiq-go"
)

func CreateMailerWorker(w *sidekiq.Worker) sidekiq.WorkerI {
	return &MailerWorker{*w, Message{}}
}

type Message struct {
	Action string `json:"action"`
	Method string `json:"method"`
	Email  string `json:"email"`
	Id     int    `json:"id"`
}

type MailerWorker struct {
	sidekiq.Worker
	Message Message
}

func (worker *MailerWorker) Work() (err error) {
	start := time.Now().UnixNano()
	json.Unmarshal([]byte(worker.Payload), &worker.Message)
	fmt.Println("Message : ", worker.Message)
	// panic("test panic")

	// err = fmt.Errorf("test")

	// 这里完成此worker的功能

	worker.LogInfo("payload Id: ", worker.Message.Id, ", time:", (time.Now().UnixNano()-start)/1000000, " ms")
	return
}
