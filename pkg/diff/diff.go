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
	"time"
)

type Diff struct {
	zk              *zookeeper.Client
	zkPrefix        []string
	zkExcludePrefix []string
	etcd            *etcd.Client
	concurrency     uint
	listWorkers     uint
	zkKeyCount      atomic.Int32
	etcdKeyCount    int
	mux             sync.Mutex
	zkKeys          sync.Map
	missed          []string
	extra           []string
	keys            chan string
	etcdKeys        map[string]bool
	workerWg        sync.WaitGroup
	stopChan        chan struct{}
}

func New(zkClient *zookeeper.Client, zkPrefix, zkExcludePrefix []string, etcd *etcd.Client, concurrency uint) *Diff {
	return &Diff{
		zk:              zkClient,
		zkPrefix:        zkPrefix,
		zkExcludePrefix: zkExcludePrefix,
		etcd:            etcd,
		concurrency:     concurrency,
		etcdKeys:        map[string]bool{},
		keys:            make(chan string, concurrency*2),
		stopChan:        make(chan struct{}),
	}
}

func (d *Diff) GetKeyCount() string {
	return d.zkKeyCount.String()
}

func (d *Diff) starDiffWorkers() {

	log.Infow("start worker",
		"concurrency", d.concurrency,
	)
	for i := uint(0); i < d.concurrency; i++ { // handle key
		go func() {
			for {
				select {
				case key := <-d.keys:
					d.handleKey(key)
				case <-d.stopChan:
					log.Debug("stop diff worker")
					return
				}
			}
		}()
	}
}

func (d *Diff) handleChildren(key string) {
	children := d.zk.List(key)
	for _, child := range children {
		newKey := filepath.Join(key, child)
		d.handleKeyRecursive(newKey)
	}
}

func (d *Diff) handleKeyRecursive(key string) {
	d.workerWg.Add(1)
	defer d.workerWg.Done()

	d.keys <- key
	children := d.zk.List(key)
	for _, child := range children {
		newKey := filepath.Join(key, child)
		d.handleKeyRecursive(newKey)
	}
}

func (d *Diff) handleKey(key string) {
	if d.shouldExclude(key) {
		return
	}
	d.mux.Lock()
	if _, ok := d.etcdKeys[key]; ok {
		delete(d.etcdKeys, key)
	} else {
		d.missed = append(d.missed, key)
	}
	d.mux.Unlock()
	d.zkKeyCount.Inc()
}

func (d *Diff) Run() {
	d.starDiffWorkers()
	var wg sync.WaitGroup
	before := time.Now()
	for _, prefix := range d.zkPrefix {
		d.etcdGetAll(prefix, &wg)
	}
	wg.Wait()
	cost := time.Since(before)
	d.etcdKeyCount = len(d.etcdKeys)
	log.Infow("fetched data from etcd",
		"duration", cost.String(),
		"count", d.etcdKeyCount,
	)

	for _, prefix := range d.zkPrefix {
		if !d.zk.Exists(prefix) {
			log.Warnw("prefix key not exsits",
				"key", prefix,
			)
			continue
		}
		d.handleKeyRecursive(prefix)
	}

	time.Sleep(time.Second)
	d.workerWg.Wait()

	close(d.stopChan)

	for k, _ := range d.etcdKeys {
		d.extra = append(d.extra, k)
	}
	d.etcdKeys = nil
}

func (d *Diff) conDo(cocurrent int, keys []string, doFunc func(string)) {
	if len(keys) == 0 {
		return
	}

	if cocurrent > len(keys) {
		cocurrent = len(keys)
	}

	var wg sync.WaitGroup
	ch := make(chan string, cocurrent)
	stop := make(chan struct{})

	for i := 0; i < cocurrent; i++ {
		go func() {
			for {
				select {
				case key := <-ch:
					doFunc(key)
					wg.Done()
				case <-stop:
					return
				}
			}
		}()
	}

	for _, key := range keys {
		wg.Add(1)
		ch <- key
	}
	wg.Wait()
	close(stop)
}

func (d *Diff) Fix() {
	// 删除 etcd 多余的 key
	d.conDo(int(d.concurrency), d.extra, func(key string) {
		d.etcd.Delete(key, false)
	})

	// 补齐 etcd 缺失的 key
	d.conDo(int(d.concurrency), d.missed, func(key string) {
		value, ok := d.zk.Get(key)
		if !ok {
			log.Errorw("try add etcd key but not exist in zk",
				"key", key,
			)
			return
		}
		d.etcd.Put(key, value)
	})
}

func (d *Diff) PrintSummary() {
	// 输出统计
	fmt.Println("zookeeper key count:", d.zkKeyCount.String())
	fmt.Println("etcd key count:", d.etcdKeyCount)
	fmt.Println("etcd missed key count:", len(d.missed))
	fmt.Println("etcd extra key count:", len(d.extra))
	fmt.Println("etcd missed keys:", d.missed)
	fmt.Println("etcd extra keys:", d.extra)
}

func (d *Diff) shouldExclude(key string) bool {
	for _, prefix := range d.zkExcludePrefix { // exclude prefix
		if strings.HasPrefix(key, prefix) {
			log.Debugw("ignore key in excluded prefix",
				"key", key,
				"excludePrefix", prefix,
			)
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
