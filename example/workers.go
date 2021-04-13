package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	sidekiq "github.com/oldfritter/sidekiq-go"

	"github.com/oldfritter/sidekiq-go/example/config"
	"github.com/oldfritter/sidekiq-go/example/initializers"
)

func main() {
	initialize()
	initializers.InitWorkers()
	StartAllWorkers()
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	closeResource()
}

func initialize() {
	setLog()
	err := os.MkdirAll("pids", 0755)
	if err != nil {
		log.Fatalf("create folder error: %v", err)
	}
	err = ioutil.WriteFile("pids/workers.pid", []byte(strconv.Itoa(os.Getpid())), 0644)
	if err != nil {
		log.Fatalf("open file error: %v", err)
	}
}

func closeResource() {
}

func StartAllWorkers() {
	for _, w := range config.AllWorkerIs {
		w.SetClient(config.Client())
		w.InitLogger()
		for i := 0; i < w.GetThreads(); i++ {

			// 启动 worker
			go func(w sidekiq.WorkerI) {
				for true {
					if d, err := sidekiq.Run(w); d && err == nil {
						time.Sleep(time.Second * 10)
					}
				}
			}(w)

		}
	}
}

func setLog() {
	err := os.Mkdir("logs", 0755)
	if err != nil {
		if !os.IsExist(err) {
			log.Fatalf("create folder error: %v", err)
		}
	}
	file, err := os.OpenFile("logs/workers.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("open file error: %v", err)
	}
	log.SetOutput(file)
}
