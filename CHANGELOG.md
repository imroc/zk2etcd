# zk2etcd 变更历史

## v0.10.0 (2021.06.26)

* etcd 操作失败自动重试，提升鲁棒性
* 移出重构前遗留的无用代码
* 支持对比 zk 和 etcd 的 key 差别 (暂时使用 `--diff` 参数，表示只进行对比，不进行同步)

## v0.9.0 (2021.06.26)

* `--zookeeper-prefix` 支持多个 prefix (逗号分隔)
* 支持 `--zookeeper-exclude-prefix` 以排除特定前缀的 key (支持多个，逗号分隔)

## v0.8.0 (2021.06.26)

* 抽离通用对象初始化逻辑(zk,etcd,logger)，方便后续与其它子命令共用(比如diff)
* 修复不配置etcd证书导致不可用问题
* 支持并发度控制，引入 `--concurrent` 参数指定同步 worker 的协程数量

## v0.7.0 (2021.06.25)

* 支持配置 etcd 证书

## v0.6.0 (2021.06.25)

* 支持同步删除，保证数据一致性
* 优化命令行参数
  * etcdAddr-->etcd-servers
  * zkAddr-->zookeeper-servers
  * zkPrefix-->zookeeper-prefix
  * 优化 description

## v0.5.0 (2021.06.25)

* etcd 逻辑独立，与 controller 解耦

## v0.4.0 (2021.06.24)

* 重构 zk client，高内聚低耦合
* 打印更多 zk 操作相关日志

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