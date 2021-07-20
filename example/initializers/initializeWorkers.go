package initializers

import (
	"io/ioutil"
	"log"
	"path/filepath"

	sidekiq "github.com/oldfritter/sidekiq-go/v1"
	"gopkg.in/yaml.v2"

	"github.com/oldfritter/sidekiq-go/v1/example/config"
	"github.com/oldfritter/sidekiq-go/v1/example/sidekiqWorkers"
)

func InitWorkers() {
	pathStr, _ := filepath.Abs("config/workers.yml")
	content, err := ioutil.ReadFile(pathStr)
	if err != nil {
		log.Fatal(err)
	}
	yaml.Unmarshal(content, &config.AllWorkers)

	config.AllWorkerIs = map[string]func(*sidekiq.Worker) sidekiq.WorkerI{
		"TreatWorker":  sidekiqWorkers.CreateTreatWorker,
		"MailerWorker": sidekiqWorkers.CreateMailerWorker,
	}
}
