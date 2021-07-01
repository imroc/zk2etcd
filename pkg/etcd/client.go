package etcd

import (
	"context"
	"crypto/tls"
	"github.com/imroc/zk2etcd/pkg/log"
	"github.com/imroc/zk2etcd/pkg/util/try"
	"github.com/prometheus/client_golang/prometheus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

var (
	EtcdOp = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zk2etcd_etcd_op_total",
			Help: "Number of the etcd operation",
		},
		[]string{"op", "status"},
	)
	EtcdOpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "zk2etcd_etcd_op_duration_seconds",
			Help:    "Duration in seconds of etcd operation",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"op", "status"},
	)
)

func init() {
	prometheus.MustRegister(EtcdOp)
	prometheus.MustRegister(EtcdOpDuration)
}

var defaultTimeout = 5 * time.Second

func timeoutContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), defaultTimeout)
	return ctx
}

type Client struct {
	*log.Logger
	servers []string
	*clientv3.Client
	tls *tls.Config
}

func NewClient(logger *log.Logger, servers []string, tls *tls.Config) *Client {
	client := &Client{
		Logger:  logger,
		servers: servers,
		tls:     tls,
	}
	err := client.init()
	if err != nil {
		logger.Errorw("init etcd client failed",
			"error", err,
		)
	}
	return client
}

func (c *Client) init() error {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   c.servers,
		DialTimeout: 5 * time.Second,
		Logger:      c.Desugar(),
		TLS:         c.tls,
	})
	if err != nil {
		return err
	}
	c.Client = client
	return nil
}

func (c *Client) put(key, value string) error {
	before := time.Now()
	c.Debugw("etcd put",
		"key", key,
		"value", value,
	)
	_, err := c.Client.Put(timeoutContext(), key, value)
	cost := time.Since(before)
	status := "success"
	if err != nil {
		status = err.Error()
		c.Errorw("etcd put failed",
			"key", key,
			"error", err.Error(),
		)
	}
	EtcdOp.WithLabelValues("put", status).Inc()
	EtcdOpDuration.WithLabelValues("put", status).Observe(float64(time.Duration(cost) / time.Second))
	return err
}

func (c *Client) do(fn func() bool) {
	try.Do(fn, 3, time.Second)
}

func (c *Client) Put(key, value string) {
	c.do(func() bool {
		err := c.put(key, value)
		if err != nil {
			return false
		}
		return true
	})
}

func (c *Client) delete(key string, prefix bool) error {
	c.Debugw("etcd delete key",
		"key", key,
		"withPrefix", prefix,
	)
	opts := []clientv3.OpOption{}
	if prefix {
		opts = append(opts, clientv3.WithPrefix())
	}
	before := time.Now()
	_, err := c.Client.Delete(timeoutContext(), key, opts...)
	cost := time.Since(before)
	status := "success"
	if err != nil {
		status = err.Error()
		c.Errorw("etcd delete failed",
			"key", key,
			"error", err.Error(),
		)
	}
	EtcdOp.WithLabelValues("delete", status).Inc()
	EtcdOpDuration.WithLabelValues("delete", status).Observe(float64(time.Duration(cost) / time.Second))
	return err
}

func (c *Client) Delete(key string, prefix bool) {
	c.do(func() bool {
		err := c.delete(key, prefix)
		if err != nil {
			return false
		}
		return true
	})
}

func (c *Client) DeleteWithPrefix(key string) {
	c.Delete(key, true)
}

func (c *Client) get(key string) (value string, ok bool, err error) {
	defer func() {
		c.Debugw("etcd get",
			"key", key,
			"value", value,
			"exist", ok,
		)
	}()

	before := time.Now()
	resp, err := c.Client.Get(timeoutContext(), key)
	cost := time.Since(before)
	status := "success"

	if err != nil {
		status = err.Error()
		c.Errorw("etcd get failed",
			"key", key,
			"error", err.Error(),
		)
	}
	EtcdOp.WithLabelValues("get", status).Inc()
	EtcdOpDuration.WithLabelValues("get", status).Observe(float64(time.Duration(cost) / time.Second))

	if err == nil && len(resp.Kvs) != 0 {
		value = string(resp.Kvs[0].Value)
		ok = true
	}
	return
}

func (c *Client) Get(key string) (value string, ok bool) {
	c.do(func() bool {
		var err error
		value, ok, err = c.get(key)
		if err != nil {
			return false
		}
		return true
	})
	return
}

func (c *Client) list(key string) ([]string, error) {
	c.Debugw("etcd list",
		"prefix", key,
	)

	before := time.Now()
	resp, err := c.Client.Get(timeoutContext(), key, clientv3.WithPrefix())
	cost := time.Since(before)

	status := "success"
	if err != nil {
		c.Errorw("etcd failed to get",
			"key", key,
			"error", err,
		)
		status = err.Error()
	}

	EtcdOp.WithLabelValues("list", status).Inc()
	EtcdOpDuration.WithLabelValues("list", status).Observe(float64(time.Duration(cost) / time.Second))

	if err != nil {
		return nil, err
	}

	keys := []string{}
	for _, kvs := range resp.Kvs {
		keys = append(keys, string(kvs.Key))
	}

	return keys, err
}

func (c *Client) ListAllKeys(key string) (keys []string) {
	c.do(func() bool {
		var err error
		keys, err = c.list(key)
		if err != nil {
			return false
		}
		return true
	})
	return
}
