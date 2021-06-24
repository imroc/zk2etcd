SHELL := /bin/bash
IMAGE := imroc/zk2etcd:0.3.0

.PHONY: build_docker
build_docker:
	docker build . -t $(IMAGE)

.PHONY: push
push:
	docker push $(IMAGE)

.PHONY: test
test:
	go run cmd/zk2etcd/main.go --zkAddr zookeeper:2181 --zkPrefix /dubbo --etcdAddr etcd:2379 --log-level info

.PHONY: build
build:
	./build.sh
