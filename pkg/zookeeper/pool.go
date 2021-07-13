package zookeeper

import (
	"github.com/go-zookeeper/zk"
	"github.com/imroc/zk2etcd/pkg/log"
	"sync"
	"time"
)

type Pool struct {
	m       sync.Mutex    // 保证多个goroutine访问时候，closed的线程安全
	conns   chan *zk.Conn //连接存储的chan
	servers []string
}

func NewPool(servers []string, size int) *Pool {
	return &Pool{
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
		return connectUntilSuccess(p.servers)
	}
}

func (p *Pool) PutConn(conn *zk.Conn) {
	p.conns <- conn
}
