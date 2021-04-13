package sidekiqWorkers

import (
	"fmt"
	"time"

	sidekiq "github.com/oldfritter/sidekiq-go"
	"github.com/oldfritter/sidekiq-go/example/config"
)

func InitializeTreatWorker() {
	for _, w := range config.AllWorkers {
		if w.Name == "TreatWorker" {
			config.AllWorkerIs["TreatWorker"] = &TreatWorker{w}
			return
		}
	}
}

type TreatWorker struct {
	sidekiq.Worker
}

func (worker *TreatWorker) Work() (err error) {
	fmt.Println("start payload: ", worker.Payload)
	start := time.Now().UnixNano()
	if err != nil {
		worker.LogError("payload: ", worker.Payload, ", time:", (time.Now().UnixNano()-start)/1000000, " ms")
		return
	}
	fmt.Println("payload: ", worker.Payload)
	// panic("test panic")

	// 这里完成此worker的功能

	worker.LogInfo("payload: ", worker.Payload, ", time:", (time.Now().UnixNano()-start)/1000000, " ms")
	return
}
