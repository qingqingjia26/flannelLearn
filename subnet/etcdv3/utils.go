package etcdv3

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"strings"

	. "github.com/qingqingjia26/flannelLearn/subnet"

	"github.com/qingqingjia26/flannelLearn/pkg/ip"
)

func getTlsConfig(etcdCert, etcdCertKey, etcdCa string) (*tls.Config, error) {
	// var etcdCert = "./ca/etcd-client.pem"
	// var etcdCertKey = "./ca/etcd-client-key.pem"
	// var etcdCa = "./ca/ca.pem"

	cert, err := tls.LoadX509KeyPair(etcdCert, etcdCertKey)
	if err != nil {
		return nil, err
	}

	caData, err := ioutil.ReadFile(etcdCa)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caData)

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}, nil
}

// func parseKey2Ipnet(key []byte, preLen int) (ip.IP4Net, error) {
// 	res := ip.IP4Net{}
// 	k := bytes.Replace(key[preLen:], []byte("-"), []byte("/"), 1)
// 	err := json.Unmarshal(k, &res)
// 	return res, err
// }

func parseKey2Ipnet(key []byte, preLen int) (ip.IP4Net, error) {
	var strSplit []string
	if strSplit = strings.Split(string(key[preLen:]), "-"); len(strSplit) != 2 {
		log.Println("the subnetRegexp doesn't run as expect:", strSplit)
		return ip.IP4Net{}, errors.New("the subnetRegexp doesn't run as expect")
	}
	ipStr, leadingOneSizeStr := strSplit[0], strSplit[1]
	_, ipNet, err := net.ParseCIDR(ipStr + "/" + leadingOneSizeStr)
	if err != nil {
		return ip.IP4Net{}, err
	}
	return ip.FromIPNet(ipNet), nil
}

func parseKeyValue(key, value []byte, preLen int) (Lease, error) {
	ipNet, err := parseKey2Ipnet(key, preLen)
	if err != nil {
		return Lease{}, err
	}
	leaseAttr := LeaseAttrs{}
	if err = json.Unmarshal(value, &leaseAttr); err != nil {
		return Lease{}, err
	}
	lease := Lease{
		Subnet: ipNet,
		Attrs:  leaseAttr,
		Ttl:    -1,
	}
	return lease, nil
}
