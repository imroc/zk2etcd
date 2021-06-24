FROM cr.imroc.cc/library/golang:latest as builder
COPY . /build
RUN cd /build/cmd/zk2etcd && go build -o /zk2etcd

FROM cr.imroc.cc/library/net-tools:latest
COPY --from=builder /zk2etcd /zk2etcd
CMD ["/zk2etcd"]