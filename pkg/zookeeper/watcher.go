package zookeeper

import (
	"github.com/go-zookeeper/zk"
	"log"
	"path/filepath"
	"time"
)

type Watcher struct {
	conn *zk.Conn
	path string
}

func New(conn *zk.Conn, path string) *Watcher {
	return &Watcher{conn: conn, path: path}
}

func (w *Watcher) Run(stop <-chan struct{}) {
	children, eventChan, shouldExit := watchUntilSuccess(w.path, w.conn)
	if shouldExit {
		return
	}
	go func(children []string) {
		for _, child := range children {
			ww := New(w.conn, filepath.Join(w.path, child))
			go ww.Run(stop)
		}
	}(children)
	for {
		select {
		case event := <-eventChan:
			if event.Type == zk.EventNodeDeleted {
				return
			}
			children, eventChan, shouldExit = watchUntilSuccess(w.path, w.conn)
			if shouldExit {
				return
			}
		case <-stop:
			return
		}
	}
}

func watchUntilSuccess(path string, conn *zk.Conn) ([]string, <-chan zk.Event, bool) {
	children, _, eventChan, err := conn.ChildrenW(path)
	//Retry until succeed
	for err != nil {
		if err == zk.ErrNoNode {
			return nil, nil, true
		}
		log.Printf("failed to watch zookeeper path %s, %v\n", path, err)
		time.Sleep(1 * time.Second)
		children, _, eventChan, err = conn.ChildrenW(path)
	}
	return children, eventChan, false
}
