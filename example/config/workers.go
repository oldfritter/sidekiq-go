package config

import (
	sidekiq "github.com/oldfritter/sidekiq-go"
)

var (
	AllWorkers  []sidekiq.Worker
	AllWorkerIs = map[string]func(*sidekiq.Worker) sidekiq.WorkerI{}
	SWI         []sidekiq.WorkerI
)
