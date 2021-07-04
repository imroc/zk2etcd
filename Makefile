SHELL := /bin/bash
IMAGE := imroc/zk2etcd:0.15.0

.PHONY: build_docker
build_docker:
	docker build . -t $(IMAGE)

.PHONY: push
push:
	docker push $(IMAGE)

.PHONY: sync
sync:
	go run cmd/zk2etcd/*.go sync --zookeeper-servers zookeeper:2181 --zookeeper-prefix /dubbo --etcd-servers etcd:2379 --log-level info --fullsync-interval 10m

.PHONY: diff
diff:
	go run cmd/zk2etcd/*.go diff --zookeeper-servers zookeeper:2181 --zookeeper-prefix /dubbo --etcd-servers etcd:2379 --log-level info --concurrent 50

.PHONY: genzk
genzk:
	go run cmd/zk2etcd/*.go genzk --zookeeper-servers zookeeper:2181

.PHONY: build
build:
	./build.sh

.PHONY: lint
lint:
	./build.sh
	rm ./bin/zk2etcd

.PHONY: clean
clean:
	rm ./bin/zk2etcd