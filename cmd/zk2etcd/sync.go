package main

import (
	"context"
	"github.com/imroc/zk2etcd/pkg/etcd"
	"github.com/imroc/zk2etcd/pkg/log"
	"github.com/imroc/zk2etcd/pkg/record"
	"github.com/imroc/zk2etcd/pkg/sync"
	"github.com/imroc/zk2etcd/pkg/zookeeper"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/cobra"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func newSyncCmd(args []string) *cobra.Command {
	var common Common
	stopChan := make(chan struct{}, 1)
	var concurrent uint
	var fullSyncInterval time.Duration
	var logOption log.Options
	var zkOption zookeeper.Options
	var etcdOption etcd.Options
	var recordOption record.Options
	var enableLeaderElect bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "sync zookeeper to etcd",

		DisableAutoGenTag: true,
		Long:              `sync data from zookeeper to etcd`,
		PreRun: func(cmd *cobra.Command, args []string) {
			log.Init(&logOption) // 初始化 logging
			zookeeper.Init(&zkOption)
			etcd.Init(&etcdOption)
			record.Init(&recordOption)
		},
		Run: func(cmd *cobra.Command, args []string) {
			run := func() {
				zkPrefixes, zkExcludePrefixes := common.GetAll()

				s := sync.New(zkPrefixes, zkExcludePrefixes, concurrent, fullSyncInterval, stopChan)
				go s.Run()

				// TODO: 实现真正优雅停止
				signalChan := make(chan os.Signal, 1)
				signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
				<-signalChan
				stopChan <- struct{}{}
			}
			if enableLeaderElect {
				namespaceFile := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
				bs, err := ioutil.ReadFile(namespaceFile)
				namespace := string(bs)
				if err != nil {
					log.Fatalw("should run in k8s if --leader-elect=true",
						"error", err.Error(),
					)
				}
				cfg, err := rest.InClusterConfig()
				if err != nil {
					log.Fatal(err.Error())
				}
				id := uuid.NewV4().String()
				client := kubernetes.NewForConfigOrDie(cfg)
				lock := &resourcelock.LeaseLock{
					LeaseMeta: metav1.ObjectMeta{
						Name:      "zk2etcd",
						Namespace: namespace,
					},
					Client: client.CoordinationV1(),
					LockConfig: resourcelock.ResourceLockConfig{
						Identity: id,
					},
				}

				// start the leader election code loop
				leaderelection.RunOrDie(context.Background(), leaderelection.LeaderElectionConfig{
					Lock: lock,
					// IMPORTANT: you MUST ensure that any code you have that
					// is protected by the lease must terminate **before**
					// you call cancel. Otherwise, you could have a background
					// loop still running and another process could
					// get elected before your background loop finished, violating
					// the stated goal of the lease.
					ReleaseOnCancel: true,
					LeaseDuration:   15 * time.Second, //租约时间
					RenewDeadline:   10 * time.Second, //更新租约的
					RetryPeriod:     2 * time.Second,  //非leader节点重试时间
					Callbacks: leaderelection.LeaderCallbacks{
						OnStartedLeading: func(ctx context.Context) {
							// we're notified when we start - this is where you would
							// usually put your code
							run()
						},
						OnStoppedLeading: func() {
							log.Errorw("leader lost",
								"id", id,
							)
							os.Exit(0)
						},
						OnNewLeader: func(identity string) {
							// we're notified when new leader elected
							if identity == id {
								log.Infow("I got the leader",
									"id", identity)
							} else {
								log.Infow("new leader elected",
									"id", identity)
							}
						},
					},
				})
			} else {
				run()
			}
		},
	}

	cmd.SetArgs(args)
	common.AddFlags(cmd.Flags())
	logOption.AddFlags(cmd.Flags())
	zkOption.AddFlags(cmd.Flags())
	etcdOption.AddFlags(cmd.Flags())
	recordOption.AddFlags(cmd.Flags())

	cmd.Flags().UintVar(&concurrent, "concurrent", 50, "the concurreny of syncing worker")
	cmd.Flags().DurationVar(&fullSyncInterval, "fullsync-interval", 5*time.Minute, "the interval of full sync, set to 0s to disable it")
	cmd.Flags().BoolVar(&enableLeaderElect, "leader-elect", true, ""+
		"Start a leader election client and gain leadership before "+
		"executing the main loop. Enable this when running replicated "+
		"components for high availability.")
	return cmd
}
