package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	sidekiq "github.com/oldfritter/sidekiq-go"

	"github.com/oldfritter/sidekiq-go/example/config"
	"github.com/oldfritter/sidekiq-go/example/initializers"
)

var (
	closeWorkersChain = make(chan int)
	app               *newrelic.Application
	err               error
)

func main() {
	initialize()
	initializers.InitWorkers()

	if app, err = newrelic.NewApplication(
		newrelic.ConfigAppName("Sidekiq-Go"),
		newrelic.ConfigLicense("fd4037a09661ecc12378f9da59b161e4a88c9c7e"),
	); err != nil {
		os.Exit(1)
	}
	if err = app.WaitForConnection(5 * time.Second); nil != err {
		fmt.Println(err)
	}

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
	err = os.MkdirAll("pids", 0755)
	if err != nil {
		log.Fatalf("create folder error: %v", err)
	}
	err = ioutil.WriteFile("pids/workers.pid", []byte(strconv.Itoa(os.Getpid())), 0644)
	if err != nil {
		log.Fatalf("open file error: %v", err)
	}
}

func closeResource() {
	app.Shutdown(time.Second * 10)
	config.CloseRedisPool()
}

func startAllWorkers() {
	for _, worker := range config.AllWorkers {
		for i := 0; i < worker.Threads; i++ {
			w := config.AllWorkerIs[worker.Name](&worker)
			w.SetConn(config.GetRedisConn())
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
	txn := app.StartTransaction(w.GetName())
	if d, err := sidekiq.Run(w); d && err == nil {
		txn.End()
		time.Sleep(time.Second * 10)
	} else {
		txn.End()
	}
	app.RecordCustomEvent(w.GetName(), map[string]interface{}{
		"color": "blue",
	})
	run(w)
}

func setLog() {
	err = os.Mkdir("logs", 0755)
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
