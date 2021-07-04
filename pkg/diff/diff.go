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
	zkPrefix        []string
	zkExcludePrefix []string
	concurrency     uint
	listWorkers     uint
	zkKeyCount      atomic.Int32
	etcdKeyCount    int
	mux             sync.Mutex
	zkKeys          sync.Map
	missed          []string
	keys            chan string
	etcdKeys        map[string]bool
	workerWg        sync.WaitGroup
	stopChan        chan struct{}
}

func New(zkPrefix, zkExcludePrefix []string, concurrency uint) *Diff {
	return &Diff{
		zkPrefix:        zkPrefix,
		zkExcludePrefix: zkExcludePrefix,
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
	children := zookeeper.List(key)
	for _, child := range children {
		newKey := filepath.Join(key, child)
		d.handleKeyRecursive(newKey)
	}
}

func (d *Diff) handleKeyRecursive(key string) {
	d.workerWg.Add(1)
	defer d.workerWg.Done()

	d.keys <- key
	children := zookeeper.List(key)
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

// 对照结果重新 check 一遍，避免 diff 期间频繁变更，递归查询结果与实际不一致
func (d *Diff) recheck() {
	etcdKeys := make(map[string]bool)
	for key, _ := range d.etcdKeys { // 如果重新check仍然是多余的key，才认为是多余的key
		_, existInEtcd := etcd.Get(key)
		if existInEtcd {
			etcdKeys[key] = true
		} else {
			d.etcdKeyCount--
		}
	}
	d.etcdKeys = etcdKeys

	missed := []string{}
	for _, key := range d.missed { // 如果重新check仍然是缺失的key，才认为是缺失的key
		_, existInEtcd := etcd.Get(key)
		//log.Infow("etcd get value",
		//	"key", key,
		//	"value", value,
		//	"existInEtcd", existInEtcd,
		//)
		if existInEtcd {
			d.etcdKeyCount++
		} else {
			//log.Info("still missing " + key)
			missed = append(missed, key)
		}
	}
	d.missed = missed
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
		if !zookeeper.Exists(prefix) {
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

	d.recheck() // 拿着重新check一遍，避免频繁变更导致结果不一致

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

func (d *Diff) getExtraKeys() []string {
	extra := []string{}
	for k, _ := range d.etcdKeys {
		extra = append(extra, k)
	}
	return extra
}

func (d *Diff) Fix() {
	// 删除 etcd 多余的 key
	d.conDo(int(d.concurrency), d.getExtraKeys(), func(key string) {
		etcd.Delete(key, false)
	})

	// 补齐 etcd 缺失的 key
	d.conDo(int(d.concurrency), d.missed, func(key string) {
		value, ok := zookeeper.Get(key)
		if !ok {
			log.Errorw("try add etcd key but not exist in zk",
				"key", key,
			)
			return
		}
		etcd.Put(key, value)
	})
	d.recheck()
}

func (d *Diff) PrintSummary() {
	// 输出统计
	extra := d.getExtraKeys()
	fmt.Println("zookeeper key count:", d.zkKeyCount.String())
	fmt.Println("etcd key count:", d.etcdKeyCount)
	fmt.Println("etcd missed key count:", len(d.missed))
	fmt.Println("etcd extra key count:", len(extra))
	fmt.Println("etcd missed keys:", d.missed)
	fmt.Println("etcd extra keys:", extra)
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
	keys := etcd.ListAllKeys(key)
	for _, key := range keys {
		if d.shouldExclude(key) {
			continue
		}
		d.etcdKeys[key] = true
	}
}
