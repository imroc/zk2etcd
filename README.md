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

## 监控 metrics

zk2etcd 通过 80 端口 (暂时写死) 暴露 metrics，路径 `/metrics`，包含以下 metrics:
* zk2etcd_etcd_op_total: etcd 操作次数统计，以下是示例 promql
  * 统计etcd写入次数: sum(irate(zk2etcd_etcd_op_total{op="put"}[2m]))
  * 统计etcd删除次数: sum(irate(zk2etcd_etcd_op_total{op="delete"}[2m]))
  * etcd 操作失败统计: sum by (status,op)(rate(zk2etcd_etcd_op_total{status!="success"}[1m]))
* zk2etcd_etcd_op_duration_seconds: etcd操作时延统计，以下是示例 promql
  * etcd p95时延: histogram_quantile(0.95, sum(rate(zk2etcd_etcd_op_duration_seconds_bucket[1m])) by (le))
  * etcd p99时延:  histogram_quantile(0.99, sum(rate(zk2etcd_etcd_op_duration_seconds_bucket[1m])) by (le))
* zk2etcd_zk_op_total: zk 操作次数统计，以下是示例 promql
  * zk 操作失败统计: sum by (status,op)(rate(zk2etcd_zk_op_total{status!="success"}[1m]))
  * zk 各类型操作 rps: sum by (op) (irate(zk2etcd_zk_op_total[2m]))
* zk2etcd_zk_op_duration_seconds: zk操作时延统计，以下是示例 promql
  * zk p95时延: histogram_quantile(0.95, sum(rate(zk2etcd_zk_op_duration_seconds_bucket[1m])) by (le))
  * zk p99时延: histogram_quantile(0.99, sum(rate(zk2etcd_zk_op_duration_seconds_bucket[1m])) by (le))
* zk2etcd_fixed_total: 全量同步 fix 时，删除或补齐 etcd 数据次数统计(通常是增量同步时丢失event导致的部分数据不同步)
  * 统计因etcd数据缺失导致的数据补齐操作次数: sum(rate(zk2etcd_fixed_total{type="put"}[2m]))
  * 统计因etcd数据多余导致的数据删除操作次数: sum(rate(zk2etcd_fixed_total{type="delete"}[2m]))
  
grafana 面板示例:

![](docs/1.png)
![](docs/2.png)

## 迭代计划

* [x] 全量同步
* [x] 增量同步
* [x] 版本管理 (version 子命令打印详细信息, 版本号/commit/buiddate 等)
* [x] 日志增强 (自定义 level + json 输出)
* [x] 并发度控制
* [x] 数据一致性 (周期性全量检测+watch delete)
* [x] 支持配置 etcd 证书
* [x] 支持配置多个 zk prefix
* [x] 检测 zk 与 etcd 数据差异的 diff 能力
* [x] 支持 prometheus 指标监控
* [x] 容灾与自愈能力
* [ ] 日志更丰富的自定义(如文件存储，事件日志)
* [ ] 支持动态加载配置(如日志级别)
* [ ] 日志记录事务id，将一系列操作串起来
* [ ] 进度统计 (需探索方案)
* [ ] 支持与 etcd 同步工具共存(不勿删其它同步工具写入的数据，需引入外部存储记录状态)
