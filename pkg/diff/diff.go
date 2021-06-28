package diff

import (
	"fmt"
	"github.com/imroc/zk2etcd/pkg/etcd"
	"github.com/imroc/zk2etcd/pkg/log"
	"github.com/imroc/zk2etcd/pkg/zookeeper"
	"go.uber.org/atomic"
	"path/filepath"
	"strings"
	"sync"
)

type Diff struct {
	*log.Logger
	zk              *zookeeper.Client
	zkPrefix        []string
	zkExcludePrefix []string
	etcd            *etcd.Client
	concurrency     uint
	zkKeyCount      atomic.Int32
	etcdKeyCount    int
	mux             sync.Mutex
	zkKeys          sync.Map
	missed          []string
	extra           []string
	keys            chan string
	etcdKeys        map[string]bool
	workerWg        sync.WaitGroup
}

func New(zkClient *zookeeper.Client, zkPrefix, zkExcludePrefix []string, etcd *etcd.Client, logger *log.Logger, concurrency uint) *Diff {
	return &Diff{
		zk:              zkClient,
		zkPrefix:        zkPrefix,
		zkExcludePrefix: zkExcludePrefix,
		Logger:          logger,
		etcd:            etcd,
		concurrency:     concurrency,
		etcdKeys:        map[string]bool{},
		keys:            make(chan string, concurrency),
	}
}

func (d *Diff) starDiffWorkers() {
	d.Info("start diff worker",
		"concurrency", d.concurrency,
	)
	for i := uint(0); i < d.concurrency; i++ {
		go func() {
			for {
				key := <-d.keys
				d.handleKey(key)
				d.workerWg.Done()
			}
		}()
	}
}

func (d *Diff) handleKey(key string) {
	if d.shouldExclude(key) {
		return
	}
	d.zkKeyCount.Inc()

	if _, ok := d.etcdKeys[key]; ok {
		d.mux.Lock()
		delete(d.etcdKeys, key)
		d.mux.Unlock()
	} else {
		d.mux.Lock()
		d.missed = append(d.missed, key)
		d.mux.Unlock()
	}

	children := d.zk.List(key)
	for _, child := range children {
		newKey := filepath.Join(key, child)
		d.workerWg.Add(1)
		d.keys <- newKey
	}
}

func (d *Diff) Run() {
	d.starDiffWorkers()
	var wg sync.WaitGroup
	for _, prefix := range d.zkPrefix {
		d.etcdGetAll(prefix, &wg)
	}
	wg.Wait()
	d.etcdKeyCount = len(d.etcdKeys)
	for _, prefix := range d.zkPrefix {
		if !d.zk.Exists(prefix) {
			d.Warnw("prefix key not exsits",
				"key", prefix,
			)
			continue
		}
		d.workerWg.Add(1)
		d.keys <- prefix
	}
	d.workerWg.Wait()
	for k, _ := range d.etcdKeys {
		d.extra = append(d.extra, k)
	}
	// 输出统计
	fmt.Println("zookeeper key count:", d.zkKeyCount.String())
	fmt.Println("etcd key count:", d.etcdKeyCount)
	fmt.Println("etcd missed keys:", d.missed)
	fmt.Println("etcd extra keys:", d.extra)
}

func (d *Diff) shouldExclude(key string) bool {
	for _, prefix := range d.zkExcludePrefix { // exclude prefix
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func (d *Diff) etcdGetAll(key string, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	keys := d.etcd.ListAllKeys(key)
	for _, key := range keys {
		if d.shouldExclude(key) {
			continue
		}
		d.etcdKeys[key] = true
	}
}
