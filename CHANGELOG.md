# zk2etcd 变更历史

## v0.3.0 (2021.06.24)

* 支持 version 子命令 (输出详细的版本信息)

## v0.2.0 (2021.06.24)

* 重构日志，使用 [zap](https://github.com/uber-go/zap) 输出。
* 默认使用 json 格式，方便日志采集
* 增加更详细的日志
* 增加 `--log-level` 参数，设置日志级别

## v0.1.0 (2021.06.24)

* 支持启动时将 zookeeper 数据全量同步到 etcd，且 watch 新增并同步到 etcd
* 支持 k8s 部署