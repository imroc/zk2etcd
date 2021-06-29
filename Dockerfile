FROM cr.imroc.cc/library/golang:latest as builder
COPY . /build
RUN cd /build && make build

FROM cr.imroc.cc/library/net-tools:latest
COPY --from=builder /build/bin/zk2etcd /bin/zk2etcd
COPY ./Shanghai /etc/localtime
CMD ["/bin/zk2etcd"]