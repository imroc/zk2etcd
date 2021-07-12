SHELL := /bin/bash
IMAGE := imroc/zk2etcd:1.1.0

.PHONY: build_docker
build_docker:
	docker buildx build --platform=linux/amd64 . -t $(IMAGE)

.PHONY: push
push:
	docker push $(IMAGE)

.PHONY: sync
sync:
	go run cmd/zk2etcd/*.go sync --zookeeper-servers zookeeper:2181 --zookeeper-exclude-prefix /dubbo/config,/roc/test --zookeeper-prefix /dubbo,/roc --etcd-servers etcd:2379 --log-level info --fullsync-interval 1m --enable-event-log

.PHONY: diff
diff:
	go run cmd/zk2etcd/*.go diff --zookeeper-servers zookeeper:2181 --zookeeper-exclude-prefix /dubbo/config,/roc/test --zookeeper-prefix /dubbo,/roc --etcd-servers etcd:2379 --log-level info --concurrent 50 --max-round 3

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

.PHONY: dt
dt:
	docker buildx build --push --platform=linux/amd64 . -t cr.imroc.cc/test/zk2etcd:latest