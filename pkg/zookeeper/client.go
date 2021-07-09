package zookeeper

import (
	"github.com/go-zookeeper/zk"
	"github.com/imroc/zk2etcd/pkg/log"
	"github.com/imroc/zk2etcd/pkg/util/try"
	"sync"
	"time"
)

type Client struct {
	connPool     *sync.Pool
	connectCount int
	lock         sync.Mutex
	servers      []string
	conn         *zk.Conn
	watchConn    *zk.Conn
}

func NewClient(servers []string) *Client {
	client := &Client{
		servers: servers,
	}
	client.connPool = &sync.Pool{
		New: func() interface{} {
			return client.connectUntilSuccess()
		},
	}
	return client
}

func EnsureExists(key string) {
	client.EnsureExists(key)
}

func (c *Client) EnsureExists(key string) {
	c.dozkn("exists", func(conn *zk.Conn) (ok bool, err error) {
		log.Debugw("zk check exists",
			"key", key,
		)
		exist, _, err := conn.Exists(key)
		if err != nil {
			log.Warnw("zk check exists failed",
				"key", key,
				"error", err.Error(),
			)
		} else if exist {
			ok = true // 确保 key 存在才结束循环
		} else {
			log.Infow("key not existed, waiting for creation",
				"key", key,
			)
		}
		return
	}, -1)
	return
}

func (c *Client) getConn() *zk.Conn {
	return c.connPool.Get().(*zk.Conn)
}

func (c *Client) putConn(conn *zk.Conn) {
	c.connPool.Put(conn)
}

func (c *Client) getWatchConn() *zk.Conn {
	if c.watchConn == nil {
		c.watchConn = c.connectUntilSuccess()
	}
	return c.watchConn
}

func ReConnect() {
	client.ReConnect()
}

func (c *Client) ReConnect() {
	count := c.connectCount
	c.lock.Lock() // TODO: 优化 edge case
	defer c.lock.Unlock()
	if c.connectCount != count {
		return
	}
	log.Info("zk try to reconnect")
	c.watchConn = c.connectUntilSuccess()
	log.Info("zk connected")
	c.connectCount++
}

func (c *Client) dozkn(op string, fn func(conn *zk.Conn) (bool, error), n int) {
	try.Do(func() bool {
		conn := c.getConn()
		defer c.putConn(conn)
		before := time.Now()
		ok, err := fn(conn)
		cost := time.Since(before)
		status := "success"
		if err != nil {
			status = err.Error()
		}
		ZKOp.WithLabelValues(op, status).Inc()
		ZKOpDuration.WithLabelValues(op, status).Observe(float64(time.Duration(cost) / time.Second))
		return ok
	}, n, time.Second)
}

// 统一封装 zk 操作，抽离连接池管理+metrics逻辑
func (c *Client) dozk(op string, fn func(conn *zk.Conn) (bool, error)) {
	c.dozkn(op, fn, 3)
}

func Exists(key string) bool {
	return client.Exists(key)
}

func (c *Client) Exists(key string) (exist bool) {
	c.dozk("exists", func(conn *zk.Conn) (ok bool, err error) {
		log.Debugw("zk check exists",
			"key", key,
		)
		exist, _, err = conn.Exists(key)
		if err != nil {
			log.Warnw("zk check exists failed",
				"key", key,
				"error", err.Error(),
			)
		} else {
			ok = true // 只要没报错就不继续
		}
		return
	})
	return

}

func Create(key string) {
	client.Create(key)
}

func (c *Client) Create(key string) {
	c.dozk("create", func(conn *zk.Conn) (ok bool, err error) {
		log.Debugw("zk create",
			"key", key,
		)
		flag := int32(0)
		acl := zk.WorldACL(zk.PermAll)
		_, err = conn.Create(key, []byte("douyudouyu"), flag, acl)
		if err != nil {
			log.Warnw("zk create key failed",
				"key", key,
				"error", err,
			)
		} else {
			ok = true
		}
		return
	})
}

func Get(key string) (value string, exist bool) {
	return client.Get(key)
}

func (c *Client) Get(key string) (value string, exist bool) {
	c.dozk("get", func(conn *zk.Conn) (ok bool, err error) {
		log.Debugw("zk get",
			"key", key,
		)
		v, _, err := conn.Get(key)
		if err != nil {
			if err == zk.ErrNoNode {
				ok = true
			} else {
				log.Warnw("zk get failed",
					"key", key,
					"error", err,
				)
			}
		} else {
			exist = true
			ok = true
			value = string(v)
		}
		return
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

func List(key string, e *log.Event) (children []string) {
	return client.List(key, e)
}

func (c *Client) List(key string, e *log.Event) (children []string) {
	c.dozk("list", func(conn *zk.Conn) (ok bool, err error) {
		e.Record("zk list",
			"key", key,
		)
		children, _, err = conn.Children(key)
		if err != nil {
			if err == zk.ErrNoNode {
				ok = true
			} else {
				log.Warnw("zk list failed",
					"key", key,
					"error", err.Error(),
				)
				e.Record("zk list failed",
					"key", key,
					"error", err.Error(),
				)
			}
		} else {
			ok = true
		}
		return
	})
	return
}

func ListW(key string, e *log.Event) (children []string, ch <-chan zk.Event) {
	return client.ListW(key, e)
}

func (c *Client) ListW(key string, e *log.Event) (children []string, ch <-chan zk.Event) {
	c.dozk("listwatch", func(conn *zk.Conn) (ok bool, err error) {
		e.Record("zk list watch",
			"key", key,
		)
		children, _, ch, err = conn.ChildrenW(key)
		if err != nil {
			if err == zk.ErrNoNode {
				ok = true
			} else {
				log.Warnw("zk list watch failed",
					"key", key,
					"error", err,
				)
				e.Record("zk list watch failed",
					"key", key,
					"error", err,
				)
			}
		} else {
			ok = true
		}
		return
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
