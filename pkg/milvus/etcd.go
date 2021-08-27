package milvus

import (
	"context"
	"sync"
	"time"

	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdEndPointHealth struct {
	Ep     string `json:"endpoint"`
	Health bool   `json:"health"`
	Error  string `json:"error,omitempty"`
}

func GetEndpointsHealth(endpoints []string) []EtcdEndPointHealth {
	hch := make(chan EtcdEndPointHealth, len(endpoints))
	var wg sync.WaitGroup
	for _, ep := range endpoints {
		wg.Add(1)
		go func(ep string) {
			defer wg.Done()

			cli, err := clientv3.New(clientv3.Config{
				Endpoints:   []string{ep},
				DialTimeout: 5 * time.Second,
			})
			if err != nil {
				hch <- EtcdEndPointHealth{Ep: ep, Health: false, Error: err.Error()}
				return
			}

			eh := EtcdEndPointHealth{Ep: ep, Health: false}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err = cli.Get(ctx, "health")
			// permission denied is OK since proposal goes through consensus to get it
			if err == nil || err == rpctypes.ErrPermissionDenied {
				eh.Health = true
			} else {
				eh.Error = err.Error()
			}

			if eh.Health {
				resp, err := cli.AlarmList(ctx)
				if err == nil && len(resp.Alarms) > 0 {
					eh.Health = false
					eh.Error = "Active Alarm(s): "
					for _, v := range resp.Alarms {
						switch v.Alarm {
						case etcdserverpb.AlarmType_NOSPACE:
							eh.Error = eh.Error + "NOSPACE "
						case etcdserverpb.AlarmType_CORRUPT:
							eh.Error = eh.Error + "CORRUPT "
						default:
							eh.Error = eh.Error + "UNKNOWN "
						}
					}
				} else if err != nil {
					eh.Health = false
					eh.Error = "Unable to fetch the alarm list"
				}

			}
			cancel()
			hch <- eh
		}(ep)
	}

	wg.Wait()
	close(hch)
	healthList := []EtcdEndPointHealth{}
	for h := range hch {
		healthList = append(healthList, h)
	}

	return healthList
}
