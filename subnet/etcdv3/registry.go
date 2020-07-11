package etcdv3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path"
	"regexp"
	"strconv"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/qingqingjia26/flannelLearn/pkg/ip"

	. "github.com/qingqingjia26/flannelLearn/subnet"
)

var subnetRegexp = regexp.MustCompile(`^((0|[1-9]\d?|1\d\d|2[0-4]\d|25[0-5])\.){3}(0|[1-9]\d?|1\d\d|2[0-4]\d|25[0-5])-(0|[1-2]\d|3[0-2])$`)
var (
	errTryAgain = errors.New("try again")
)

type Registry interface {
	getNetworkConfig(ctx context.Context) (string, error)
	getSubnets(ctx context.Context) ([]Lease, uint64, error)
	getSubnet(ctx context.Context, sn ip.IP4Net) (*Lease, uint64, error)
	createSubnet(ctx context.Context, sn ip.IP4Net, attrs *LeaseAttrs, ttl int64) (int64, error)
	updateSubnet(ctx context.Context, sn ip.IP4Net, attrs *LeaseAttrs, ttl int64, asof uint64) (int64, error)
	deleteSubnet(ctx context.Context, sn ip.IP4Net) error
	watchSubnets(ctx context.Context, since uint64) (Event, uint64, error)
	watchSubnet(ctx context.Context, since uint64, sn ip.IP4Net) (Event, uint64, error)
}

type EtcdConfig struct {
	Endpoints []string
	Keyfile   string
	Certfile  string
	CAFile    string
	Prefix    string
	Username  string
	Password  string
}

type Etcdv3Registry struct {
	config        *EtcdConfig
	client        *clientv3.Client
	keyLeaseIdMap map[string]clientv3.LeaseID
}

func IpnetIntersect(n1, n2 ip.IP4Net) bool {
	return n2.Contains(n1.IP) || n1.Contains(n2.IP)
}

func NewEtcdv3Registry(cfg *EtcdConfig) (*Etcdv3Registry, error) {
	var (
		err error
	)
	clientv3Cfg := clientv3.Config{
		Endpoints: cfg.Endpoints,
		Username:  cfg.Username,
		Password:  cfg.Password,
	}
	r := &Etcdv3Registry{}
	r.config = cfg
	if cfg.Certfile != "" && cfg.Keyfile != "" && cfg.CAFile != "" {
		clientv3Cfg.TLS, err = getTlsConfig(cfg.Certfile, cfg.Keyfile, cfg.CAFile)
		if err != nil {
			return nil, err
		}
	}

	if r.client, err = clientv3.New(clientv3Cfg); err != nil {
		log.Println(err)
		return nil, err
	}
	r.keyLeaseIdMap = make(map[string]clientv3.LeaseID)
	return r, nil
}

func (r *Etcdv3Registry) getNetworkConfig(ctx context.Context) (string, error) {
	var (
		resp *clientv3.GetResponse
		err  error
	)
	client := r.client
	kv := clientv3.NewKV(client)

	key := path.Join(r.config.Prefix, "config")
	if resp, err = kv.Get(ctx, key); err != nil {
		return "", err
	}
	if resp.Count <= 0 {
		return "", errors.New(fmt.Sprintf("please set %s", key))
	}
	return string(resp.Kvs[0].Value), err
}

func (r *Etcdv3Registry) getLeaseId(client *clientv3.Client, Ttl int64) (clientv3.LeaseID, error) {
	lease := clientv3.NewLease(client)
	var leaseId clientv3.LeaseID
	if leaseGrantResp, err := lease.Grant(context.TODO(), Ttl); err != nil {
		log.Println(err)
		return leaseId, err
	} else {
		leaseId = leaseGrantResp.ID
		return leaseId, nil
	}
}

func (r *Etcdv3Registry) getLeaseIdTtl(client *clientv3.Client, leaseId int64) (int64, error) {
	lease := clientv3.NewLease(client)
	if leaseRes, err := lease.TimeToLive(context.TODO(), clientv3.LeaseID(leaseId)); err != nil {
		return -1, err
	} else {
		return leaseRes.TTL, nil
	}
}

func (r *Etcdv3Registry) createSubnet(ctx context.Context, sn ip.IP4Net, attrs *LeaseAttrs, Ttl int64) (int64, error) {
	// if allSubNet, _, err := r.getSubnets(context.TODO()); err != nil {
	// 	return -1, err
	// } else {
	// 	for _, subnet := range allSubNet {
	// 		if subnet.Subnet.Contains(sn.IP) || sn.Contains(subnet.Subnet.IP) {
	// 			log.Println("input subnet:", sn, " subnet in etcd:", subnet.Subnet, "this new subnet overlaps with existing subnets")
	// 			return -1, errors.New("this new subnet overlaps with existing subnets")
	// 		}
	// 	}
	// }

	leaseId, err := r.getLeaseId(r.client, Ttl)
	if err != nil {
		return -1, err
	}
	value, err := json.Marshal(attrs)
	if err != nil {
		log.Println(attrs, err)
		return -1, err
	}

	kv := clientv3.NewKV(r.client)

	key := path.Join(r.config.Prefix, "subnets", sn.IP.String()+"-"+strconv.Itoa(int(sn.PrefixLen)))
	// var putResp *clientv3.PutResponse
	if _, err = kv.Put(context.TODO(), key, string(value), clientv3.WithLease(leaseId)); err != nil {
		return -1, err
	}
	// log.Println(putResp)
	return Ttl, nil
}

func (r *Etcdv3Registry) Lock(ctx context.Context, run func() error) error {
	var err error
	cli := r.client
	// ctx, _ := context.WithTimeout(context.Background(), r.config.LockTimeout)
	session, err := concurrency.NewSession(cli, concurrency.WithContext(ctx))
	if err != nil {
		return err
	}
	defer session.Close()
	mutex := concurrency.NewMutex(session, "/my-lock/")
	err = mutex.Lock(context.Background())
	if err != nil {
		return nil
	}
	defer mutex.Unlock(context.Background())
	err = run()
	if err != nil {
		return nil
	}
	return nil
}

func (r *Etcdv3Registry) getSubnets(ctx context.Context) ([]Lease, uint64, error) {
	var (
		resp *clientv3.GetResponse
		err  error
	)
	client := r.client
	kv := clientv3.NewKV(client)

	if resp, err = kv.Get(ctx, path.Join(r.config.Prefix, "subnets"), clientv3.WithPrefix()); err != nil {
		return nil, 0, err
	}
	allLease := []Lease{}
	for i, _ := range resp.Kvs {
		preLen := len(path.Join(r.config.Prefix, "subnets")) + 1
		key, value := resp.Kvs[i].Key, resp.Kvs[i].Value
		leaseId := resp.Kvs[i].Lease
		if subnetRegexp.Match(key[preLen:]) {
			lease, err := parseKeyValue(key, value, preLen)
			if err != nil {
				log.Println("ignore key:", string(key), " value:", string(value), err)
				continue
			}
			Ttl, err := r.getLeaseIdTtl(client, leaseId)
			lease.Expiration = time.Now().Local().Add(time.Duration(Ttl) * time.Second)
			if err != nil {
				log.Println("getLeaseIdTtl failed:", err)
				lease.Expiration = time.Now()
			}
			allLease = append(allLease, lease)
		}
		// log.Printf("Key is s %s  Value is %s \n", key[preLen:], resp.Kvs[i].Value)
	}
	return allLease, uint64(resp.Header.Revision), nil
}

func (r *Etcdv3Registry) getSubnet(ctx context.Context, sn ip.IP4Net) (*Lease, uint64, error) {
	var (
		resp *clientv3.GetResponse
		err  error
	)
	client := r.client
	kv := clientv3.NewKV(client)

	key := path.Join(r.config.Prefix, "subnets", sn.IP.String()+"-"+strconv.Itoa(int(sn.PrefixLen)))
	if resp, err = kv.Get(ctx, key); err != nil {
		return nil, 0, err
	}
	if resp.Count == 0 {
		errMsg := "can't get this IPNet in etcd:" + sn.String()
		log.Println(errMsg)
		return nil, 0, errors.New(errMsg)
	}

	leaseAttr := LeaseAttrs{}
	if err = json.Unmarshal(resp.Kvs[0].Value, &leaseAttr); err != nil {
		log.Println(err)
		return nil, 0, err
	}
	lease := Lease{
		Subnet: sn,
		Attrs:  leaseAttr,
	}
	Ttl, err := r.getLeaseIdTtl(client, resp.Kvs[0].Lease)
	lease.Expiration = time.Now().Local().Add(time.Duration(Ttl) * time.Second)
	if err != nil {
		log.Println("getLeaseIdTtl failed:", err)
		lease.Expiration = time.Now()
	}
	return &lease, uint64(resp.Header.Revision), err
}

func (r *Etcdv3Registry) watchSubnet(ctx context.Context, since uint64, sn ip.IP4Net) (Event, uint64, error) {
	client := r.client
	watcher := clientv3.NewWatcher(client)
	watchRespChan := watcher.Watch(ctx, path.Join(r.config.Prefix, "subnets", sn.IP.String()+"-"+strconv.Itoa(int(sn.PrefixLen))), clientv3.WithRev(int64(since)+1))
	// 处理kv变化事件
	// var watchResp WatchResponse
	select {
	case <-ctx.Done():
		return Event{}, 0, ctx.Err()
	case watchResp := <-watchRespChan:
		{
			for _, event := range watchResp.Events {
				switch event.Type {
				case mvccpb.PUT:
					preLen := len(path.Join(r.config.Prefix, "subnets")) + 1
					key, value := event.Kv.Key, event.Kv.Value
					if subnetRegexp.Match(key[preLen:]) {
						lease, err := parseKeyValue(key, value, preLen)
						if err != nil {
							log.Println("key:", string(key), " value:", string(value), err)
							continue
						}
						Ttl, err := r.getLeaseIdTtl(client, event.Kv.Lease)
						lease.Expiration = time.Now().Local().Add(time.Duration(Ttl) * time.Second)
						if err != nil {
							return Event{}, 0, err
						}
						return Event{Type: EventAdded, Lease: lease}, uint64(watchResp.Header.Revision), nil
					}
				case mvccpb.DELETE:
					preLen := len(path.Join(r.config.Prefix, "subnets")) + 1
					key, value := event.Kv.Key, event.Kv.Value
					if subnetRegexp.Match(key[preLen:]) {
						ipNet, err := parseKey2Ipnet(key, preLen)
						if err != nil {
							log.Println("key:", string(key), " value:", string(value), err)
							continue
						}
						lease := Lease{Subnet: ipNet}
						return Event{Type: EventRemoved, Lease: lease}, uint64(watchResp.Header.Revision), nil
					}
				}
			}
		}
	}
	return Event{}, 0, errors.New("sth wrong in watchSubnets")
}

func (r *Etcdv3Registry) watchSubnets(ctx context.Context, since uint64) (Event, uint64, error) {
	client := r.client
	watcher := clientv3.NewWatcher(client)
	watchRespChan := watcher.Watch(ctx, path.Join(r.config.Prefix, "subnets"), clientv3.WithPrefix())
	// 处理kv变化事件
	select {
	case <-ctx.Done():
		return Event{}, 0, ctx.Err()
	case watchResp := <-watchRespChan:
		{
			for _, event := range watchResp.Events {
				switch event.Type {
				case mvccpb.PUT:
					preLen := len(path.Join(r.config.Prefix, "subnets")) + 1
					key, value := event.Kv.Key, event.Kv.Value
					if subnetRegexp.Match(key[preLen:]) {
						lease, err := parseKeyValue(key, value, preLen)
						if err != nil {
							log.Println("key:", string(key), " value:", string(value), err)
							continue
						}
						Ttl, err := r.getLeaseIdTtl(client, event.Kv.Lease)
						lease.Expiration = time.Now().Local().Add(time.Duration(Ttl) * time.Second)
						if err != nil {
							return Event{}, 0, err
						}
						return Event{Type: EventAdded, Lease: lease}, uint64(watchResp.Header.Revision), nil
					}
				case mvccpb.DELETE:
					preLen := len(path.Join(r.config.Prefix, "subnets")) + 1
					key, value := event.Kv.Key, event.Kv.Value
					if subnetRegexp.Match(key[preLen:]) {
						ipNet, err := parseKey2Ipnet(key, preLen)
						if err != nil {
							log.Println("key:", string(key), " value:", string(value), err)
							continue
						}
						lease := Lease{Subnet: ipNet}
						return Event{Type: EventRemoved, Lease: lease}, uint64(watchResp.Header.Revision), nil
					}
				}
			}
		}
	}
	return Event{}, 0, errors.New("sth wrong in watchSubnets")
}

func (r *Etcdv3Registry) deleteSubnet(ctx context.Context, sn ip.IP4Net) error {
	var (
		// resp *clientv3.DeleteResponse
		err error
	)
	client := r.client
	kv := clientv3.NewKV(client)

	key := path.Join(r.config.Prefix, "subnets", sn.IP.String()+"-"+strconv.Itoa(int(sn.PrefixLen)))
	if _, err = kv.Delete(ctx, key); err != nil {
		return err
	}
	return nil
}

func (r *Etcdv3Registry) updateSubnet(ctx context.Context, sn ip.IP4Net, attrs *LeaseAttrs, Ttl int64, Asof uint64) (int64, error) {
	return r.createSubnet(ctx, sn, attrs, Ttl)
}
