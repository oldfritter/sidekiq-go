package main

import (
	"context"
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

var (
	closeWorkersChain = make(chan int)
)

func main() {
	initialize()
	initializers.InitWorkers()
	startAllWorkers()
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutdown Server ...")
	go recycle()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	select {
	case <-closeWorkersChain:
		cancel()
	case <-ctx.Done():
		cancel()
	}

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
	config.CloseClient()
}

func startAllWorkers() {
	for _, worker := range config.AllWorkers {
		for i := 0; i < worker.Threads; i++ {
			w := config.AllWorkerIs[worker.Name](&worker)
			w.SetClient(config.Client())
			w.InitLogger()
			w.RegisterQueue()

			// 启动 worker
			go func(w sidekiq.WorkerI) {
				run(w)
			}(w)
			config.SWI = append(config.SWI, w)
			log.Println("started: ", w.GetName(), "[", i, "]")

		}
	}
}

func run(w sidekiq.WorkerI) {
	if d, err := sidekiq.Run(w); d && err == nil {
		time.Sleep(time.Second * 10)
	}
	run(w)
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

func recycle() {
	for i, w := range config.SWI {
		log.Println("stoping: ", w.GetName(), "[", i, "]")
		config.SWI[i].Stop()
	}
	for i, _ := range config.SWI {
		config.SWI[i].Recycle()
	}
	closeWorkersChain <- 1
}
