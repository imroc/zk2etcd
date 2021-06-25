SHELL := /bin/bash
IMAGE := imroc/zk2etcd:0.7.0

.PHONY: build_docker
build_docker:
	docker build . -t $(IMAGE)

.PHONY: push
push:
	docker push $(IMAGE)

.PHONY: test
test:
	go run cmd/zk2etcd/*.go --zookeeper-servers zookeeper:2181 --zookeeper-prefix /dubbo --etcd-servers etcd:2379 --log-level debug --etcd-cacert=./debug-roc/ca.crt --etcd-cert=./debug-roc/cert.pem --etcd-key=./debug-roc/key.pem

.PHONY: build
build:
	./build.sh

.PHONY: lint
lint:
	./build.sh
	rm ./bin/zk2etcd