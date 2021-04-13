package config

import (
	sidekiq "github.com/oldfritter/sidekiq-go"
)

var (
	AllWorkers  []sidekiq.Worker
	AllWorkerIs = make(map[string]sidekiq.WorkerI)
)
