package etcdv3

import (
	"context"
	. "etcdLearn/subnet"
	"log"
	"testing"
)

// func TestGetNetworkConfig(t *testing.T) {
// 	cfg := EtcdConfig{Endpoints: []string{"127.0.0.1:2379"}}
// 	r, err := NewEtcdv3Registry(&cfg)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	marshalStr, err := r.getNetworkConfig()
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	marshalStr1, err := json.Marshal(cfg)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	if marshalStr != string(marshalStr1) {
// 		t.Error("getNetworkConfig doesn't equal cfg")
// 	}
// 	log.Println(marshalStr, string(marshalStr1))
// 	log.Println(r.client)
// }

// func TestCreateSubnet(t *testing.T) {
// 	log.Println(time.Time{})
// }

// func TestGetSubnets(t *testing.T) {
// 	key := "/coreos.com/network/subnets/10.0.95.0-24"
// 	cfg := EtcdConfig{Endpoints: []string{"127.0.0.1:2379"}}
// 	r, err := NewEtcdv3Registry(&cfg)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	ctx, _ := context.WithTimeout(context.Background(), 4*time.Second)
// 	err = r.Lock(ctx, func() error {
// 		// _, _, err := r.getSubnets(ctx)
// 		_, ipNet, err := net.ParseCIDR("10.0.0.0/24")
// 		ip4Net := ip.FromIPNet(ipNet)
// 		attr := &LeaseAttrs{
// 			PublicIP:    ip4Net.IP,
// 			BackendType: "vxlan",
// 			BackendData: []byte("test"),
// 		}
// 		exp, err := r.createSubnet(ctx, ip4Net, attr, 24)
// 		if err != nil {
// 			log.Println(exp, err)
// 			return err
// 		}
// 		return nil
// 	})
// 	log.Println(err)
// }

// func TestCreateSubnets(t *testing.T) {
// 	log.SetFlags(log.Lshortfile)
// 	cfg := EtcdConfig{Endpoints: []string{"127.0.0.1:2379"}, Prefix: "/coreos.com/network"}
// 	r, err := NewEtcdv3Registry(&cfg)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	ctx, _ := context.WithTimeout(context.Background(), 4*time.Second)
// 	err = r.Lock(ctx, func() error {
// 		_, ipNet, err := net.ParseCIDR("10.0.1.0/24")
// 		tmpStr, err := json.Marshal("string")
// 		publicIp, _, err := net.ParseCIDR("192.168.2.1/0")

// 		attr := &LeaseAttrs{
// 			PublicIP:    publicIp,
// 			BackendType: "vxlan",
// 			BackendData: json.RawMessage(tmpStr),
// 		}
// 		exp, err := r.createSubnet(ctx, ipNet, attr, 100)
// 		log.Println(exp, err)
// 		if err != nil {
// 			return err
// 		}
// 		// allLease, err := r.getSubnets(ctx)
// 		// log.Println(allLease)
// 		lease, err := r.getSubnet(ctx, ipNet)
// 		log.Println(lease)
// 		return err
// 	})
// 	log.Println(err)
// }

// func TestWatchSubnets(t *testing.T) {
// 	var wg sync.WaitGroup

// 	log.SetFlags(log.Lshortfile)
// 	cfg := EtcdConfig{Endpoints: []string{"127.0.0.1:2379"}, Prefix: "/coreos.com/network"}
// 	r, err := NewEtcdv3Registry(&cfg)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	wg.Add(1)
// 	go func() {
// 		for {
// 			event, _, err := r.watchSubnets(context.TODO(), 0)
// 			if event.Type == EventAdded {
// 				log.Println("add subnet:", event, err)
// 			} else {
// 				log.Println("del subnet:", event, err)
// 			}
// 		}
// 	}()
// 	go func() {
// 		for i := 0; i < 20; i++ {
// 			time.Sleep(1 * time.Second)
// 			ctx, _ := context.WithTimeout(context.Background(), 4*time.Second)
// 			err = r.Lock(ctx, func() error {
// 				_, ipNet, err := net.ParseCIDR("10.0." + strconv.Itoa(i) + ".0/24")
// 				tmpStr, err := json.Marshal("string")
// 				publicIp, _, err := net.ParseCIDR("192.168.2.1/0")

// 				attr := &LeaseAttrs{
// 					PublicIP:    ip.FromIP(publicIp),
// 					BackendType: "vxlan",
// 					BackendData: json.RawMessage(tmpStr),
// 				}
// 				_, err = r.createSubnet(ctx, ip.FromIPNet(ipNet), attr, 100)
// 				// log.Println(exp, err)
// 				if err != nil {
// 					return err
// 				}
// 				// allLease, err := r.getSubnets(ctx)
// 				// log.Println(allLease)
// 				// lease, err := r.getSubnet(ctx, ipNet)
// 				// log.Println(lease)
// 				return err
// 			})
// 		}
// 	}()
// 	go func() {
// 		for i := 0; i < 20; i++ {
// 			time.Sleep(2 * time.Second)
// 			ctx, _ := context.WithTimeout(context.Background(), 1*time.Minute)
// 			err = r.Lock(ctx, func() error {
// 				_, ipNet, err := net.ParseCIDR("10.0." + strconv.Itoa(i) + ".0/24")
// 				err = r.deleteSubnet(ctx, ip.FromIPNet(ipNet))
// 				if err != nil {
// 					return err
// 				}
// 				return err
// 			})
// 		}
// 	}()
// 	wg.Wait()
// }
func TestGetNetworkConfig(t *testing.T) {
	lease := Lease{}
	log.Println(lease)
	log.SetFlags(log.Lshortfile)
	cfg := EtcdConfig{Endpoints: []string{"127.0.0.1:2379"}, Prefix: "/coreos.com/network"}
	r, err := NewEtcdv3Registry(&cfg)
	if err != nil {
		t.Error(err)
	}
	value, err := r.getNetworkConfig(context.TODO())
	if err != nil || value != `{"NetWork":"10.0.0.0/16", "SubnetMin": "10.0.1.0", "SubnetMax": "10.0.200.0","Backend": {"Type": "vxlan"}}` {
		t.Error(err, "getNetworkConfig failed")
	} else {
		log.Println("getNetworkConfig successfully")
	}
}
