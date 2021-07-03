package zookeeper

import (
	"github.com/go-zookeeper/zk"
	"github.com/imroc/zk2etcd/pkg/log"
	"github.com/imroc/zk2etcd/pkg/util/try"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
	"time"
)

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

type Client struct {
	connectCount int
	lock         sync.Mutex
	servers   []string
	conn      *zk.Conn
	watchConn *zk.Conn
}

func NewClient(servers []string) *Client {
	client := &Client{
		servers: servers,
	}
	return client
}

func (c *Client) EnsureExists(key string) {
	// check path exsitence
	exists := false
	for !exists {
		var err error
		exists, _, err = c.getConn().Exists(key)
		if err != nil {
			log.Errorw("failed to check path existence",
				"error", err,
				"key", key,
			)
			continue
		}
		if !exists {
			log.Warnw("zookeeper path doesn't exist, wait until it's created",
				"key", key,
			)
		}
		time.Sleep(time.Second * 2)
	}
}

func (c *Client) getConn() *zk.Conn {
	if c.conn == nil {
		c.conn = c.connectUntilSuccess()
	}
	return c.conn
}

func (c *Client) getWatchConn() *zk.Conn {
	if c.watchConn == nil {
		c.watchConn = c.connectUntilSuccess()
	}
	return c.watchConn
}

func (c *Client) ReConnect() {
	count := c.connectCount
	c.lock.Lock() // TODO: 优化 edge case
	defer c.lock.Unlock()
	if c.connectCount != count {
		return
	}
	c.watchConn = c.connectUntilSuccess()
	c.connectCount++
}

func (c *Client) exists(key string) (exist bool, err error) {
	log.Debugw("check zk exsit",
		"key", key,
	)
	before := time.Now()
	exist, _, err = c.getConn().Exists(key)
	cost := time.Since(before)
	status := "success"
	if err != nil {
		status = err.Error()
	}
	ZKOp.WithLabelValues("exists", status).Inc()
	ZKOpDuration.WithLabelValues("exists", status).Observe(float64(time.Duration(cost) / time.Second))
	return
}

func (c *Client) do(fn func() bool) {
	try.Do(fn, 3, time.Second)
}

func (c *Client) Exists(key string) (exist bool) {
	c.do(func() bool {
		var err error
		exist, err = c.exists(key)
		if err != nil {
			return false
		}
		return true
	})
	return
}

func (c *Client) Create(key string) {
	flag := int32(0)
	acl := zk.WorldACL(zk.PermAll)
	_, err := c.getConn().Create(key, []byte("douyudouyu"), flag, acl)
	if err != nil {
		log.Errorw("zk create key failed",
			"error", err,
			"key", key,
		)
	}
}

func (c *Client) get(key string) (value string, exist bool, err error) {
	log.Debugw("zk get",
		"key", key,
	)
	before := time.Now()
	v, _, err := c.getConn().Get(key)
	cost := time.Since(before)
	status := "success"
	exist = true
	if err != nil {
		if err == zk.ErrNoNode {
			err = nil
			exist = false
			log.Warnw("zk get but not exist",
				"key", key,
			)
		} else {
			status = err.Error()
			log.Errorw("zk get failed",
				"key", key,
				"error", err,
			)
		}
	}
	value = string(v)
	ZKOp.WithLabelValues("get", status).Inc()
	ZKOpDuration.WithLabelValues("get", status).Observe(float64(time.Duration(cost) / time.Second))
	return
}

func (c *Client) Get(key string) (value string, exist bool) {
	c.do(func() bool {
		var err error
		value, exist, err = c.get(key)
		if err != nil {
			return false
		}
		return true
	})
	return
}

// Delete 暂时不用
//func (c *Client) Delete(key string) {
//	log.Debugw("zk delete",
//		"key", key,
//	)
//	_, s, err := c.getConn().Get(key)
//	for err != nil {
//		log.Errorw("zk delete failed",
//			"key", key,
//			"error", err,
//		)
//		metrics.Interrupt.WithLabelValues(err.Error()).Inc()
//		time.Sleep(time.Second)
//	}
//	// TODO: 提升健壮性，处理删除冲突，进行重试
//	err = c.getConn().Delete(key, s.Version)
//	for err != nil {
//		log.Errorw("zk delete failed",
//			"key", key,
//			"error", err,
//		)
//		metrics.Interrupt.WithLabelValues(err.Error()).Inc()
//		time.Sleep(time.Second)
//	}
//}

func (c *Client) list(key string) (children []string, err error) {
	log.Debugw("zk list",
		"key", key,
	)
	before := time.Now()
	children, _, err = c.getConn().Children(key)
	cost := time.Since(before)
	status := "success"
	if err != nil && err != zk.ErrNoNode {
		log.Errorw("zk list failed",
			"key", key,
			"error", err,
		)
		status = err.Error()
	}
	ZKOp.WithLabelValues("list", status).Inc()
	ZKOpDuration.WithLabelValues("list", status).Observe(float64(time.Duration(cost) / time.Second))
	return
}

func (c *Client) List(key string) (children []string) {
	c.do(func() bool {
		var err error
		children, err = c.list(key)
		if err != nil {
			return false
		}
		return true
	})
	return
}

func (c *Client) listW(key string) (children []string, ch <-chan zk.Event, err error) {
	log.Debugw("zk list watch",
		"key", key,
	)
	before := time.Now()
	children, _, ch, err = c.getWatchConn().ChildrenW(key)
	cost := time.Since(before)
	status := "success"
	if err != nil && err != zk.ErrNoNode {
		log.Errorw("failed to list watch zookeeper",
			"key", key,
			"error", err,
		)
		status = err.Error()
	}
	ZKOp.WithLabelValues("list watch", status).Inc()
	ZKOpDuration.WithLabelValues("list watch", status).Observe(float64(time.Duration(cost) / time.Second))
	return
}

func (c *Client) ListW(key string) (children []string, ch <-chan zk.Event) {
	c.do(func() bool {
		var err error
		children, ch, err = c.listW(key)
		if err != nil {
			return false
		}
		return true
	})
	return
}

func (c *Client) connect() (conn *zk.Conn, err error) {
	log.Debugw("zk connect",
		"servers", c.servers,
	)
	conn, _, err = zk.Connect(c.servers, time.Second*10)
	return
}

func (c *Client) connectUntilSuccess() *zk.Conn {
	conn, err := c.connect()
	for err != nil { // 如果连接失败，一直重试，直到成功
		log.Errorw("failed to connect to zooKeeper, retrying...",
			"error", err,
			"servers", c.servers,
		)
		time.Sleep(time.Second)
		conn, err = c.connect()
	}
	return conn
}
