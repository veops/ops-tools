package discover

import (
	"errors"
	"github.com/sirupsen/logrus"
	"math/big"
	"sort"
	//"sort"
	"strings"
	//"time"

	gsnmp "github.com/gosnmp/gosnmp"

	. "netTopology/internal"
)

func Layer3IpLinks(target string, snmpClient *gsnmp.GoSNMP) (horizontal, vertical map[string]int64, err error) {
	horizontal = map[string]int64{}
	vertical = map[string]int64{}
	routeType, er := FetchItems("walk", target, []string{OLibrary.IpRouteType}, snmpClient)
	if er != nil {
		err = errors.New(StringConcat("fetch item failed for ", target, OLibrary.IpRouteType))
		return
	}
	routeNextHop, er := FetchItems("walk", target, []string{OLibrary.IpRouteNextHop}, snmpClient)
	if er != nil {
		err = errors.New(StringConcat("fetch item failed for ", target, OLibrary.IpRouteNextHop))
		return
	}
	routeIfIndex, er := FetchItems("walk", target, []string{OLibrary.IpRouteIfIndex}, snmpClient)
	if er != nil {
		err = errors.New(StringConcat("fetch item failed for ", target, OLibrary.IpRouteIfIndex))
		return
	}
	nextHop := map[string]string{}
	for _, v := range routeNextHop {
		nextHop[strings.TrimPrefix(v.Oid, OLibrary.IpRouteNextHop+".")] = v.DataValue.(string)
	}
	ifIndex := map[string]int64{}
	for _, v := range routeIfIndex {
		ifIndex[strings.TrimPrefix(v.Oid, OLibrary.IpRouteIfIndex+".")] = v.DataValue.(*big.Int).Int64()
	}
	var key string
	for _, v := range routeType {
		key = strings.TrimPrefix(v.Oid, OLibrary.IpRouteType+".")
		// 有时候下一跳指向本地地址，或者防火墙virtual-if*; virtual-if0为根防火墙
		if v1, ok := IpIdent.Get(nextHop[key]); ok && v1.(string) == target {
			continue
		}
		if v.DataValue.(*big.Int).Int64() == 3 {
			vertical[key] = ifIndex[key]
		} else if v.DataValue.(*big.Int).Int64() == 4 {
			horizontal[key] = ifIndex[key]
		}
	}

	newHorizontal := map[string]int64{}
	newVertical := map[string]int64{}

	for k, v := range vertical {
		v1 := Final(k, nextHop)
		if v1 != k && v1 != "127.0.0.1" && v1 != "127.0.0.0" {
			newVertical[v1] = v
		}
	}

	for k, v := range horizontal {
		v1 := Final(k, nextHop)
		if v1 != k && v1 != "127.0.0.1" && v1 != "127.0.0.0" {
			newHorizontal[v1] = v
		}
	}
	vertical = newVertical
	horizontal = newHorizontal
	return
}

// L3Topology
// 访问当前路由器的路由表,对每个路由表项
// ipRouteType等于 indirect(4)，将路由表中ipRouteNextHop的内容不重复地加入邻居列表中
// ipRouteType等于 direct(3)，把ipRouteDest和 ipRouteMask不重复地放到本地相连子网队列中。
func L3Topology(target string, graph *TopologyGraph, recursive bool) {
	defer Topology3SyncWait.Done()
	if !ValidInnerIp(target) {
		logrus.Warnf("invalid intranet ip for %s", target)
		return
	}
	if _, ok := L3Nodes.Load(target); ok {
		return
	}
	if v, ok := IpIdent.Get(target); ok {
		target = v.(string)
	}
	snmpClient, er := NewSnmp(target)
	if er != nil {
		logrus.Infof("New snmp client for %s failed %s", target, er.Error())
		return
	} else {
		defer snmpClient.Conn.Close()
	}
	var (
		localIps, peerIps []string
	)
	arp := DiscoverIps(target, snmpClient)
	if arp == nil || arp.Target == "" {
		logrus.Info("get empty ip for ", target)
		return
	}
	target = arp.Target
	IpEntOfDevice.Store(target, arp.SelfIps)
	_ = peerIps
	if target == "" {
		logrus.Info("get self ip failed for ", target)
		return
	}

	horizontal, vertical, er := Layer3IpLinks(target, snmpClient)
	if er != nil {
		logrus.Warnf("Layer3IpLinks %s failed: %v", target, er)
		return
	}
	for k := range vertical {
		localIps = append(localIps, k)
	}
	{
		if _, ok := L3Nodes.LoadOrStore(target, false); !ok {
			topology3Lock.Lock()
			graph.Nodes = append(graph.Nodes, &Node{
				Name:   target,
				Type:   GetNodeType(target).String(),
				Locals: arp.Subnets,
				Ips:    arp.SelfIps,
			})
			topology3Lock.Unlock()
		}
	}
	for ip, ifIndex := range horizontal {
		Topology3SyncWait.Add(1)
		go func(ip string, ifIndex int64) {
			snmpCli, er := NewSnmp(ip)
			if er != nil {
				logrus.Infof("New snmp client[%s] failed %s", ip, er.Error())
				Topology3SyncWait.Done()
				return
			} else {
				defer snmpCli.Conn.Close()
			}
			tmpArp := DiscoverIps(ip, snmpCli)
			if tmpArp.Target == "" {
				Topology3SyncWait.Done()
				return
			}
			if _, ok := L3Nodes.LoadOrStore(tmpArp.Target, false); !ok {
				topology3Lock.Lock()
				graph.Nodes = append(graph.Nodes, &Node{
					Name:   tmpArp.Target,
					Type:   GetNodeType(ip).String(),
					Locals: tmpArp.Subnets,
					Ips:    tmpArp.SelfIps,
				})
				topology3Lock.Unlock()
				if recursive {
					L3Topology(tmpArp.Target, graph, recursive)
				} else {
					Topology3SyncWait.Done()
				}
			} else {
				Topology3SyncWait.Done()
			}
		}(ip, ifIndex)
	}
}

// SubnetIps 获取本机IP、连接的子网
func SubnetIps(target string, snmpClient *gsnmp.GoSNMP) (subnetIps, subnetMask []string, er error) {
	r1, err := FetchItems("walk", target, []string{OLibrary.IpAdEntAddr}, snmpClient)
	if err != nil {
		er = err
		return
	}
	r2, err := FetchItems("walk", target, []string{OLibrary.IpAdEntNetMask}, snmpClient)
	if err != nil {
		er = err
		return
	}
	var (
		ips map[string]string
		obj string
	)
	for _, v := range r1 {
		obj = strings.TrimPrefix(v.Oid, OLibrary.IpAdEntAddr)
		ips[obj] = v.DataValue.(string)
	}

	for _, v := range r2 {
		obj = strings.TrimPrefix(v.Oid, OLibrary.IpAdEntNetMask)
		if ips[obj] != "" && ips[obj] != "127.0.0.1" && ips[obj] != "127.0.0.0" {
			subnetIps = append(subnetIps, ips[obj])
			subnetMask = append(subnetMask, v.DataValue.(string))
		}
	}
	return
}

// DiscoverIps 发现网络IP以及自身IP
// localIps 本机ip列表； peerIps 对端ip列表
func DiscoverIps(target string, snmpClient *gsnmp.GoSNMP) (result *Arp) {
	logrus.Info("DiscoverIps for ARP ", target)
	result = &Arp{}
	if v, ok := DiscoverIpsCache.Get(target); ok {
		return v.(*Arp)
	}
	defer func() {
		logrus.Info("DiscoverIps for ARP end: ", target)
		DiscoverIpsCache.SetDefault(target, result)
	}()
	r0, er := FetchItems("walk", target, []string{OLibrary.IpAdEntAddr}, snmpClient)
	if er != nil {
		logrus.Warnf("fetch %s %s failed %v", target, OLibrary.IpAdEntAddr, er)
		return
	}
	//r1, er := FetchItems("walk", target, []string{OLibrary.IpNetToMediaNetAddress}, snmpClient)
	//if er != nil {
	//	fmt.Println(er)
	//	return
	//}
	//r2, er := FetchItems("walk", target, []string{OLibrary.IpNetToMediaType}, snmpClient)
	//if er != nil {
	//	fmt.Println(er)
	//	return
	//}
	r3, er := FetchItems("walk", target, []string{OLibrary.IpRouteDest}, snmpClient)
	if er != nil {
		logrus.Warnf("fetch %s %s failed %v", target, OLibrary.IpRouteDest, er)
		return
	}
	r4, er := FetchItems("walk", target, []string{OLibrary.IpAdEntNetMask}, snmpClient)
	if er != nil {
		logrus.Warnf("fetch %s %s failed %v", target, OLibrary.IpAdEntNetMask, er)
		return
	}
	r5, er := FetchItems("walk", target, []string{OLibrary.IpRouteType}, snmpClient)
	if er != nil {
		logrus.Warnf("fetch %s %s failed %v", target, OLibrary.IpRouteType, er)
		return
	}
	r6, er := FetchItems("walk", target, []string{OLibrary.IpRouteMask}, snmpClient)
	if er != nil {
		logrus.Warnf("fetch %s %s failed %v", target, OLibrary.IpRouteMask, er)
		return
	}
	var (
		//ips   = map[string]string{}
		masks = map[string]string{}
		//obj   string
	)
	{
		for _, v := range r4 {
			masks[strings.TrimPrefix(v.Oid, OLibrary.IpAdEntNetMask)] = v.DataValue.(string)
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
			//ip := StringConcat(v.DataValue.(string), "/", masks[strings.TrimPrefix(v.Oid, OLibrary.IpAdEntAddr)])
			result.SelfIps = append(result.SelfIps, v.DataValue.(string))
			//if !StringInSlice(ip, result.Subnets) {
			//	result.Subnets = append(result.Subnets, ip)
			//}
		}
		for _, v := range result.SelfIps {
			v = strings.Split(v, "/")[0]
			if _, er := NewSnmp(v); er == nil {
				result.Target = v
				break
			} else if PingHost(v) {
				result.Target = v
				continue
			}
		}
		for _, v := range result.SelfIps {
			v = strings.Split(v, "/")[0]
			IpIdent.SetDefault(v, result.Target)
		}
	}
	//{
	//	for _, v := range r1 {
	//		obj = strings.TrimPrefix(v.Oid, OLibrary.IpNetToMediaNetAddress)
	//		ips[obj] = v.DataValue.(string)
	//	}
	//	for _, v := range r2 {
	//		obj = strings.TrimPrefix(v.Oid, OLibrary.IpNetToMediaType)
	//		switch v.DataValue.(*big.Int).String() {
	//		case "3":
	//			result.PeerIps = append(result.PeerIps, ips[obj])
	//		case "4", "1":
	//			if ips[obj] != "" && ips[obj] != "127.0.0.1" && ips[obj] != "127.0.0.0" {
	//				ip := StringConcat(ips[obj], "/", masks["."+strings.SplitN(obj, ".", 3)[2]])
	//				if !StringInSlice(ip, result.Subnets) {
	//					result.Subnets = append(result.Subnets, ip)
	//				}
	//			}
	//		}
	//	}
	//}
	{
		dest := map[string]string{}
		routeType := map[string]int64{}
		for _, v := range r3 {
			dest[strings.TrimPrefix(v.Oid, OLibrary.IpRouteDest)] = v.DataValue.(string)
		}
		for _, v := range r5 {
			routeType[strings.TrimPrefix(v.Oid, OLibrary.IpRouteType)] = v.DataValue.(*big.Int).Int64()
		}
		for _, v := range r6 {
			key := strings.TrimPrefix(v.Oid, OLibrary.IpRouteMask)
			if v1, ok := routeType[key]; ok {
				if v1 == int64(3) {
					if !strings.HasPrefix(dest[key], "127.") &&
						dest[key] != "0.0.0.0" &&
						!StringInSlice(dest[key], result.SelfIps) {
						result.Subnets = append(result.Subnets, StringConcat(dest[key], "/", v.DataValue.(string)))
					}
				} else if v1 == int64(4) {
					result.PeerIps = append(result.PeerIps, StringConcat(dest[key], "/", v.DataValue.(string)))
				}
			}
		}
	}
	Routes.Store(result.Target, result)
	return
}
