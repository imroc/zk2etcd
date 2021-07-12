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
	ZKConn = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zk2etcd_zk_conn_total",
			Help: "Number of the zk connection",
		},
		[]string{},
	)
	ZKConnect = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zk2etcd_zk_connect_total",
			Help: "Number of the zk connect times",
		},
		[]string{},
	)
)

func init() {
	prometheus.MustRegister(ZKOp)
	prometheus.MustRegister(ZKOpDuration)
	prometheus.MustRegister(ZKConn)
	prometheus.MustRegister(ZKConnect)
}
