package zookeeper

import (
	"github.com/go-zookeeper/zk"
	"github.com/imroc/zk2etcd/pkg/log"
	"sync"
	"time"
)

type Pool struct {
	conns   chan *zk.Conn //连接存储的chan
	size    int
	count   int
	lock    sync.Mutex
	servers []string
}

func NewPool(servers []string, size int) *Pool {
	return &Pool{
		size:    size,
		servers: servers,
		conns:   make(chan *zk.Conn, size),
	}
}

func connect(servers []string) (conn *zk.Conn, err error) {
	ZKConnect.WithLabelValues().Inc()
	log.Debugw("zk connect",
		"servers", servers,
	)
	conn, _, err = zk.Connect(servers, time.Second*10)
	return
}

func connectUntilSuccess(servers []string) *zk.Conn {
	ZKConn.WithLabelValues().Inc()
	conn, err := connect(servers)
	for err != nil { // 如果连接失败，一直重试，直到成功
		log.Errorw("failed to connect to zooKeeper, retrying...",
			"error", err,
			"servers", servers,
		)
		time.Sleep(time.Second)
		conn, err = connect(servers)
	}
	return conn
}

func (p *Pool) GetConn() *zk.Conn {
	select {
	case c := <-p.conns:
		return c
	default:
		var conn *zk.Conn
		p.lock.Lock()
		if p.count < p.size {
			conn = connectUntilSuccess(p.servers)
			p.count++
		} else {
			conn = <-p.conns
		}
		p.lock.Unlock()
		return conn
	}
}

func (p *Pool) PutConn(conn *zk.Conn) {
	p.conns <- conn
}
