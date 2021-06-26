package main

import (
	"fmt"
	"go.uber.org/atomic"
	"path/filepath"
	"strings"
	"sync"
)

var (
	zkKeyCount   atomic.Int32
	etcdKeyCount int
	mux          sync.Mutex
	zkKeys       sync.Map
	missed       []string
	extra        []string
	keys         chan string
	etcdKeys     map[string]bool
	workerWg     sync.WaitGroup
)

func starDiffWorkers() {
	for i := uint(0); i < concurrent; i++ {
		go func() {
			for {
				key := <-keys
				handleKey(key)
				workerWg.Done()
			}
		}()
	}
}

func handleKey(key string) {
	if shouldExclude(key) {
		return
	}
	zkKeyCount.Inc()

	if _, ok := etcdKeys[key]; ok {
		mux.Lock()
		delete(etcdKeys, key)
		mux.Unlock()
	} else {
		mux.Lock()
		missed = append(missed, key)
		mux.Unlock()
	}

	children := zkClient.List(key)
	for _, child := range children {
		newKey := filepath.Join(key, child)
		workerWg.Add(1)
		keys <- newKey
	}
}

func diff() {
	etcdKeys = map[string]bool{}
	keys = make(chan string, concurrent)
	starDiffWorkers()
	var wg sync.WaitGroup
	for _, prefix := range zkPrefixes {
		etcdGetAll(prefix, &wg)
	}
	wg.Wait()
	etcdKeyCount = len(etcdKeys)
	for _, prefix := range zkPrefixes {
		if !zkClient.Exists(prefix) {
			logger.Warnw("prefix key not exsits",
				"key", prefix,
			)
			continue
		}
		workerWg.Add(1)
		keys <- prefix
	}
	workerWg.Wait()
	for k, _ := range etcdKeys {
		extra = append(extra, k)
	}
	// 输出统计
	fmt.Println("zookeeper key count:", zkKeyCount.String())
	fmt.Println("etcd key count:", etcdKeyCount)
	fmt.Println("etcd missied keys:", missed)
	fmt.Println("etcd extra keys:", extra)
}

func shouldExclude(key string) bool {
	for _, prefix := range zkExcludePrefixes { // exclude prefix
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func etcdGetAll(key string, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	keys := etcdClient.ListAllKeys(key)
	for _, key := range keys {
		if shouldExclude(key) {
			continue
		}
		etcdKeys[key] = true
	}
}
