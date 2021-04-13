package initializers

import (
	"io/ioutil"
	"log"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/oldfritter/sidekiq-go/example/config"
)

func init() {
	path_str, _ := filepath.Abs("config/redis.yml")
	content, err := ioutil.ReadFile(path_str)
	if err != nil {
		log.Fatal(err)
		return
	}
	err = yaml.Unmarshal(content, &config.RedisCluster)
	if err != nil {
		log.Fatal(err)
		return
	}
	config.InitRedis()
}
