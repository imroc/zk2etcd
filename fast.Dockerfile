FROM cr.imroc.cc/library/net-tools:latest
COPY ./bin/zk2etcd /bin/zk2etcd
COPY ./Shanghai /etc/localtime
CMD ["/bin/zk2etcd"]