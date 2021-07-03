package zookeeper

import "github.com/prometheus/client_golang/prometheus"

var (
	ZKOp = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zk2etcd_zk_op_total",
			Help: "Number of the zk operation",
		},
		[]string{"op", "status"},
	)
	ZKOpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "zk2etcd_zk_op_duration_seconds",
			Help:    "Duration in seconds of zk operation",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"op", "status"},
	)
)

func init() {
	prometheus.MustRegister(ZKOp)
	prometheus.MustRegister(ZKOpDuration)
}