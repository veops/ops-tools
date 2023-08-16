package internal

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	gsnmp "github.com/gosnmp/gosnmp"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"io"
	"sort"
	"strings"
	"time"
)

const (
	defaultCommunity   = "public"
	defaultCommunityV2 = "publicv2"
)

var (
	TargetStatus   = cache.New(time.Minute, time.Minute)
	CommunityMap   = cache.New(time.Hour*24, time.Hour)
	SnmpVersionMap = cache.New(time.Hour*24, time.Hour)
	ResultCache    = cache.New(time.Hour, time.Minute*15)
	IpNodeTypeMaps = cache.New(time.Hour*24*7, time.Hour)
	// IpIdent ip映射，当一个设备有多个ip时,统一为一个IP采集数据
	IpIdent = cache.New(time.Hour*24*7, time.Hour*24)
)

type SnmpResp struct {
	Domain    string      `json:"domain"`
	Oid       string      `json:"oid"`
	Key       string      `json:"key"`
	DataType  string      `json:"data_type"`
	DataValue interface{} `json:"data_value"`
	Raw       interface{} `json:"raw"`
}

func NewSnmp(target string) (*gsnmp.GoSNMP, error) {
	if v, ok := TargetStatus.Get(target); ok {
		if !v.(bool) {
			return nil, fmt.Errorf("connect error")
		}
	}
	if !ValidInnerIp(target) {
		return nil, fmt.Errorf("invalid intranet ip for %s", target)
	}
	c := defaultCommunityV2
	if v, ok := CommunityMap.Get(target); ok {
		c = v.(string)
	}
	gVersion := gsnmp.Version2c
	if v, ok := SnmpVersionMap.Get(target); ok {
		switch v {
		case "1":
			gVersion = gsnmp.Version1
		case "3":
			// TODO
			gVersion = gsnmp.Version3
		}
	}
	s := &gsnmp.GoSNMP{
		Target:             target, // eg:"172.18.0.2"
		Port:               161,
		Community:          c,
		Version:            gVersion,
		Timeout:            time.Second * 3,
		Retries:            1,
		Transport:          "udp",
		ExponentialTimeout: false,
	}
	er := s.Connect()
	defer func() {
		if er != nil {
			logrus.Warnf("try new snmp %s failed %v", target, er)
		} else {
			logrus.Infof("try new snmp %s success", target)
		}
	}()
	if er == nil {
		if _, er1 := SnmpWalk(s, "walk", OLibrary.IpForwarding); er1 != nil {
			er = er1
		}
	}
	if er != nil {
		s.Community = func(c string) string {
			if c == defaultCommunity {
				return defaultCommunityV2
			} else {
				return defaultCommunity
			}
		}(c)
		er = s.Connect()
		if er == nil {
			if _, er1 := SnmpWalk(s, "walk", OLibrary.IpForwarding); er1 != nil {
				er = er1
			}
		}
		if er != nil {
			TargetStatus.SetDefault(target, false)
			return s, er
		}
	}
	s.Timeout = time.Second * 15
	s.Retries = 1
	CommunityMap.SetDefault(target, s.Community)
	TargetStatus.SetDefault(target, true)
	return s, nil
}

func FetchItems(action, target string, oids []string, snmp *gsnmp.GoSNMP, args ...interface{}) (res []*SnmpResp, err error) {
	if len(oids) == 0 {
		return res, fmt.Errorf("%s", "oid is none")
	}
	key := fmt.Sprintf("%s-%s-%s", action, target, strings.Join(oids, "-"))
	if tmp, exist := ResultCache.Get(key); exist {
		return tmp.([]*SnmpResp), nil
	}
	if snmp == nil {
		snmp, err = NewSnmp(target)
		if err != nil {
			return
		}
		defer snmp.Conn.Close()
	}

	res, err = SnmpWalk(snmp, action, oids...)
	cacheTime := -2
	if len(args) > 0 {
		cacheTime = args[0].(int)
	}
	if err == nil {
		if cacheTime == -2 {
			if strings.HasSuffix(key, OLibrary.IpNetToMediaPhysAddress) {
				ResultCache.Set(key, res, time.Minute*30)
			} else {
				ResultCache.Set(key, res, time.Second*10)
			}
		} else if cacheTime > 0 {
			ResultCache.Set(key, res, time.Second*time.Duration(cacheTime))
		} else {
			ResultCache.SetDefault(key, res)
		}
	}
	return
}

func SnmpCheck(hostName string, oids ...string) (string, bool) {
	cli, err := NewSnmp(hostName)
	if err != nil {
		//fmt.Println("snmp check failed:", hostName, err)
		return "", false
	}
	cli.Timeout = time.Second
	defer cli.Conn.Close()
	if len(oids) == 0 {
		res, er := SnmpWalk(cli, "walk", OLibrary.IpForwarding)
		if er != nil {
			return "", false
		} else if len(res) > 0 {
			return fmt.Sprintf("%v", res[0].DataValue), true
		} else {
			return "", true
		}
	} else {
		_, er := cli.Get(oids)
		return "", er == nil
	}
}

func GetNodeType(node string) NodeType {
	if nt, ok := IpNodeTypeMaps.Get(node); ok {
		return nt.(NodeType)
	}
	nt := NTOther
	defer func() {
		IpNodeTypeMaps.SetDefault(node, nt)
	}()
	// 只有mac地址，无法通过snmp获取数据，暂且标记为server
	if len(strings.Split(node, ".")) != 4 {
		nt = NTServer
		return nt
	}
	if v, supportSnmp := SnmpCheck(node); supportSnmp {
		if v == "2" {
			nt = NTServer
		} else {
			if _, ok := SnmpCheck(node, fmt.Sprintf("%s.0.0.0.0", OLibrary.IpRouteIfIndex)); ok {
				nt = NTRouter
			} else if _, ok := SnmpCheck(node, OLibrary.Dot1dBaseBridgeAddress); ok {
				nt = NTSwitch
			}
		}
	} else {
		if PingHost(node) {
			nt = NTServer
		}
	}
	return nt
}

// IpIdentGet 针对非服务器，获取标识ip,
func IpIdentGet(node string, snmp *gsnmp.GoSNMP) (ip string, ips []string, err error) {
	ips = []string{}
	if v, ok := IpIdent.Get(node); ok {
		return v.(string), ips, nil
	}
	if snmp == nil {
		snmp, er := NewSnmp(node)
		if er != nil {
			err = er
			return
		} else {
			defer snmp.Conn.Close()
		}
	}
	r0, er := FetchItems("walk", node, []string{OLibrary.IpAdEntAddr}, snmp)
	if er != nil {
		err = er
		return
	}
	sort.Slice(r0, func(i, j int) bool {
		return InetAtoi(r0[i].DataValue.(string)) < InetAtoi(r0[j].DataValue.(string))
	})
	for _, v := range r0 {
		if v.DataValue.(string) == "127.0.0.1" ||
			v.DataValue.(string) == "0.0.0.0" ||
			v.DataValue.(string) == "127.0.0.0" {
			continue
		}
		ips = append(ips, v.DataValue.(string))
		if _, er := NewSnmp(v.DataValue.(string)); er == nil {
			IpIdent.SetDefault(node, v.DataValue.(string))
			ip = v.DataValue.(string)
			break
		} else if PingHost(v.DataValue.(string)) {
			IpIdent.SetDefault(node, v.DataValue.(string))
			ip = v.DataValue.(string)
			continue
		}
	}
	return
}

func MD5(str string) string {
	h := md5.New()
	_, _ = io.WriteString(h, str)
	return hex.EncodeToString(h.Sum(nil))
}
