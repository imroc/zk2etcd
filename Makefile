SHELL := /bin/bash
IMAGE := imroc/zk2etcd:0.1.0

.PHONY: build
build:
	docker build . --no-cache -t $(IMAGE)

.PHONY: push
push:
	docker push $(IMAGE)

.PHONY: test
test:
	go run cmd/zk2etcd/main.go --zkAddr zookeeper:2181 --zkPrefix /dubbo --etcdAddr etcd:2379

.PHONY: check
check:
	go build -o ./output/zk2etcd ./cmd/zk2etcd

