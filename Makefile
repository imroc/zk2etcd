SHELL := /bin/bash
IMAGE := imroc/zk2etcd:0.9.0

.PHONY: build_docker
build_docker:
	docker build . -t $(IMAGE)

.PHONY: push
push:
	docker push $(IMAGE)

.PHONY: test
test:
	go run cmd/zk2etcd/*.go --zookeeper-servers zookeeper:2181 --zookeeper-prefix /dubbo --etcd-servers etcd:2379 --log-level debug

.PHONY: build
build:
	./build.sh

.PHONY: lint
lint:
	./build.sh
	rm ./bin/zk2etcd