package sync

import "github.com/prometheus/client_golang/prometheus"

var (
	Fix = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zk2etcd_fixed_total",
			Help: "Number of fixed operation in full sync",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(Fix)
}
