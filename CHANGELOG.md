# zk2etcd 变更历史

## v1.2.2 (2021.07.15)

* 为 fullsync 丰富事件日志

## v1.2.1 (2021.07.13)

* 修复 log api 空指针问题
* 增加 log 子命令，封装 log api 调用，方便动态调整日志级别
* 增加 api 说明文档

## v1.2.0 (2021.07.13)

* 重新实现 zk 连接池，修复 goroutine/内存 泄漏问题
* 优化 diff
* metrics 增加 zk2etcd_zk_conn_total (连接池的zk连接数量)
* metrics 增加 zk2etcd_zk_connect_total (zk connect 次数)
* 支持 pprof
* 增加 /log 的 http api，实现动态调整日志级别

## v1.1.0 (2021.07.09)

* 支持 event log，使用 `--enable-event-log` 开启 (带 `"type": "event"` 字段)。
* event log 将相关联的日志使用相同 traceId 串起来，方便运维与问题排查。

## v1.0.4 (2021.07.08)

* 支持自定义 redis 用户名。

## v1.0.3 (2021.07.08)

* 支持配置 redis 密码。

## v1.0.2 (2021.07.07)

* 支持多轮 diff，`--max-round` 指定最大几轮 diff，`--round-interval` 指定每轮 diff 间隔时长。
* 完善文档: 增加核心原理与方案的说明。

## v1.0.1 (2021.07.06)

* 支持 `--redis-server` 可选，不指定表示不启用与其它同步工具共存的特性。

## v1.0.0 (2021.07.06)

* metrics 增加 zk2etcd_fixed_total 指标，统计全量同步发生 fix 时的增删统计。
* 修复增量同步时 exclude key 逻辑不生效问题。
* 支持与其它同步工具共存，避免勿删其它工具写入的 key，引入 redis 存储状态，在 sync 和 diff 子命令增加 `--redis-server` 参数指定 redis 地址。
* 完善运维手册

## v0.16.0 (2021.07.04)

* 增强 diff: 递归 diff 判断之后立即拿着结果 recheck 一遍，避免频繁变更导致结果幻象。
* 修复 diff --fix 在 fix 之后打印的结果不一致问题。

## v0.15.0 (2021.07.04)

* 大量重构: zk,etcd,logging 相关逻辑在单独的包中高内聚，引入全局默认对象，初始化+flag+option逻辑封装，减少复杂度和bug风险，降低继续叠加功能的成本。
* 同步循环引入debounce逻辑，避免短时间内频繁ChildrenChanged导致频繁sync，也避免 ChildrenChanged 与 NodeDeleted 并行导致不一致问题。

## v0.14.0 (2021.07.02)

* 支持 prometheus metrics
* 增强自愈能力
* 一些重构，为后续继续优化铺垫基础

## v0.13.0 (2021.06.29)

* `zk2etcd diff` 支持 `--fix` 参数，指示是否订正数据差异

## v0.12.0 (2021.06.29)

* `zk2etcd sync` 支持定期全量检测来订正，通过 `--fullsync-interval` 自定义周期，默认 `5m`
* 增强 watch 机制，确保新增的 node 也能被 watch
* 全量检测并订正的机制改用 diff 判断 zk 与 etcd 之间的差异 (zk 与 etcd v3 存储机制不同，zk 树状而 etcd 是扁平，无法使用类似 list children 的方式来对比子节点差异)
* 优化判断逻辑
* 丰富debug日志
* 打包时区文件到镜像(默认东八区)

## v0.11.0 (2021.06.28)

* 抽离同步逻辑到单独子命令 `zk2etcd sync`
* 抽离diff逻辑到单独子命令 `zk2etcd diff`
* diff与sync定时打印状态/进度
* 添加 `zk2etcd genzk` 子命令，用于生成 zk 测试数据
* 增强单 key 同步逻辑，确保zk中不存在但etcd中存在时删除etcd中数据
* 删 etcd 中 key 时，如果下面有子节点，也都全部删除
* 定时打印已同步/已比较的进度及其耗时信息
* 修复diff hang死的bug
* 修改 sync 默认并发 20 --> 50

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