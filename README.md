# zk2etcd

zk2etcd 是一款同步 zookeeper 数据到 etcd 的工具

## 项目背景

在云原生大浪潮下，业务都逐渐上 k8s，许多业务以前使用 zookeeper 作为注册中心，现都逐渐倾向更加贴近云原生的 etcd。

在业务向云原生迁移改造的过程中，可能需要将 zookeeper 中注册的数据同步到 etcd，且可能需要两者长期共存，实时同步增量数据。

本项目就是为了解决 zookeeper 数据同步到 etcd 而生。

## 部署

[examples](examples) 下提供部署到 k8s 的 yaml 示例。

## 变更历史

参考 [CHANGELOG](CHANGELOG.md)

## 迭代计划

* [x] 全量同步
* [x] 增量同步
* [x] 版本管理 (version 子命令打印详细信息, 版本号/commit/buiddate 等)
* [x] 日志增强 (自定义 level + json 输出)
* [ ] 数据一致性 (周期性全量检测+watch delete)
* [ ] 支持配置 etcd 证书
* [ ] 支持配置多个 zk prefix
* [ ] 可观测增强
* [ ] 并发度控制
* [ ] 进度统计 (需探索方案)
* [ ] 容灾与自愈能力
* [ ] 检测 zk 与 etcd 数据差异的 diff 能力