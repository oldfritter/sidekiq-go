# SneakerWorker for Golang


### Dependencies 依赖
* Redis

### Usage 使用方法
1.在你的项目中的config目录下创建以下文件

```
workers.yml
```
This is a [example](https://github.com/oldfritter/sidekiq-go/blob/master/example/config).

2.workers

[example  示例](https://github.com/oldfritter/sidekiq-go/blob/master/example/sidekiqWorkers)

3.main

[example  示例](https://github.com/oldfritter/sidekiq-go/blob/master/example/workers.go)

### workers.yml配置说明
```
---
- name: TreatWorker						  # worker的名称
  queue: sidekiq:example:treat  # 消息进入的queue
  log: logs/treat_worker.log		# 日志
  threads: 1							      # 并发处理数量
```
