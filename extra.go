package sidekiq

// 附加：记录log
func (worker *Worker) LogInfo(text ...interface{}) {
	worker.logger.SetPrefix("INFO: " + worker.GetName() + " ")
	worker.logger.Println(text)
}

func (worker *Worker) LogDebug(text ...interface{}) {
	worker.logger.SetPrefix("DEBUG: " + worker.GetName() + " ")
	worker.logger.Println(text)
}

func (worker *Worker) LogError(text ...interface{}) {
	worker.logger.SetPrefix("ERROR: " + worker.GetName() + " ")
	worker.logger.Println(text)
}
