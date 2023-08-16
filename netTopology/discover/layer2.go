package discover

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"math/big"
	"strconv"
	"strings"
	"sync"

	gsnmp "github.com/gosnmp/gosnmp"

	. "netTopology/internal"
)

type IfIndexInfo struct {
	IfIndex int64
	// ToIps对接的所有IP
	ToIps []string
	// Neighbor 对接的IP
	Neighbor string
	Port     string
	// Des 接口别名
	Des string
}

type Subnet struct {
	Name string `json:"name"`
	// Router 网关路由器
	Router string `json:"router"`
	// Routers 子网内的交换机
	Routers map[string]string `json:"routers"`
	// Switches 子网内的交换机
	Switches map[string]string `json:"switches"`
	// Hosts 子网内的主机
	Hosts map[string]string `json:"hosts"`
}

var (
	IfIndexMapIpRecord = sync.Map{}
	NodeInfo           = sync.Map{}
	NodeLinks          = sync.Map{}
	commonLocker       = sync.RWMutex{}
	//ifIndexPortMap     = sync.Map{}
)

type Subnets struct {
	sync.RWMutex
	M map[string]*Subnet
}

func (s *Subnets) Get(name string) *Subnet {
	s.RLock()
	defer s.RUnlock()
	return s.M[name]
}

func (s *Subnets) Add(fieldType NodeType, name, value string) {
	s.Lock()
	defer s.Unlock()
	if _, ok := s.M[name]; !ok {
		s.M[name] = &Subnet{
			Name:     name,
			Switches: map[string]string{},
			Hosts:    map[string]string{},
			Routers:  map[string]string{},
		}
	}
	switch fieldType {
	case NTRouter:
		s.M[name].Router = value
		s.M[name].Routers[value] = ""
	case NTSwitch:
		s.M[name].Switches[value] = ""
	case NTServer:
		s.M[name].Hosts[value] = ""
	}
}

func (s *Subnets) Clear() {
	s.Lock()
	defer s.Unlock()
	s.M = map[string]*Subnet{}
}

func (s *Subnets) Range(f func(k, v any) bool) {
	s.RLock()
	defer s.RUnlock()
	for k, v := range s.M {
		if !f(k, v) {
			break
		}
	}
}

func ScanTotalSubnetsAndNeighbor(graph *TopologyGraph) (map[string]map[string]int64, map[string]map[string]string) {
	SubnetsVar.Clear()
	IfIndexMapIpRecord = sync.Map{}
	// step1 扫描出各个子网内的交换机、路由器，以及整个arp表
	{
		wg := sync.WaitGroup{}
		ch := make(chan struct{}, 1000)
		Routes.Range(func(key, value any) bool {
			for _, objs := range [][]string{value.(*Arp).Subnets, value.(*Arp).PeerIps} {
				for _, v := range objs {
					if _, ok := SubnetsScanned.LoadOrStore(v, struct{}{}); ok {
						continue
					}
					ips, er := CidrHosts(v)
					if er != nil {
						logrus.Warn(er)
						continue
					}
					for _, v1 := range ips {
						if !ValidInnerIp(v1) {
							logrus.Info("invalid intranet ips", v1)
							break
						}
						wg.Add(1)
						ch <- struct{}{}
						go func(ip, cidr string, wg *sync.WaitGroup) {
							defer func() {
								wg.Done()
								<-ch
							}()
							nt := GetNodeType(ip)
							SubnetsVar.Add(nt, cidr, ip)
							switch nt {
							case NTSwitch, NTRouter:
								_, _ = ScanArp(ip, nil)
								Topology3SyncWait.Add(1)
								L3Topology(ip, graph, false)
								Topology3SyncWait.Wait()
							}
						}(v1, v, &wg)
					}
				}
			}
			return true
		})
		wg.Wait()
	}
	logrus.Debug("end scan arp......")

	// step2 针对子网的route,switch 获取接口映射
	logrus.Info("start scan links")
	portMapIfIndex := map[string]map[string]int64{}
	portMapDes := map[string]map[string]string{}
	var f1 = func(ip string) {
		ident, r, er := IfIndexMapIp(ip, nil)
		if er != nil || len(r) == 0 {
			return
		}
		portMapIfIndex[ident] = map[string]int64{}
		portMapDes[ident] = map[string]string{}
		for ifIndex, v1 := range r {
			portMapIfIndex[ident][v1.Port] = ifIndex
			portMapDes[ident][v1.Port] = v1.Des
		}
		NodeInfo.Store(ident, r)
		NodeLink(ident, nil)
	}
	{
		SubnetsVar.Range(func(k, v any) bool {
			for ip := range v.(*Subnet).Routers {
				f1(ip)
			}
			for ip := range v.(*Subnet).Switches {
				f1(ip)
			}
			return true
		})
	}
	r := map[string]StpInfo{}
	NodeLinks.Range(func(key, value any) bool {
		v := value.(StpInfo)
		r[key.(string)] = v
		return true
	})
	var links []*Link
	for _, v := range r {
		nodeFrom := v.Port
		if v1, ok := portMapDes[v.AgentIp]; ok {
			if v2, ok := v1[v.Port]; ok {
				nodeFrom = v2
			}
		} else if v1, ok := portMapIfIndex[v.AgentIp]; ok {
			if v2, ok := v1[v.Port]; ok {
				nodeFrom = fmt.Sprintf("%v", v2)
			}
		}
		gh := &Link{
			NodeFromName: v.AgentIp,
			NodeFrom:     nodeFrom,
			NodeToName:   v.DesignAddress,
		}
		if v1, ok := r[v.DesignAddress]; ok {
			if v2, _, er := IpIdentGet(v1.AgentIp, nil); er == nil {
				gh.NodeToName = v2
			} else {
				gh.NodeToName = v1.AgentIp
			}
		} else {
			logrus.Warn("unmatched mac address", v.DesignAddress)
		}
		links = append(links, gh)
	}
	graph.Links = links
	//// mac to ips
	//for i, v := range graph.Nodes {
	//	for i1, v1 := range v.DesIps {
	//		ips := []string{}
	//		for _, v2 := range v1 {
	//			if v3, ok := MacIps.Load(v2); ok {
	//				ips = append(ips, v3.([]string)...)
	//			}
	//		}
	//		ips = utils.RmDupElement(ips)
	//		sort.Strings(ips)
	//		graph.Nodes[i].DesIps[i1] = ips
	//		fmt.Println("ips:", ips)
	//	}
	//}
	return portMapIfIndex, portMapDes
}

func ifIndexMapIpSwitch(node string, data map[int64]*IfIndexInfo, snmp *gsnmp.GoSNMP) {
	macIfIndex := map[string]int64{}
	portIfIndex := map[string]int64{}
	ifIndexPort := map[int64]string{}
	ifIndexDes := map[int64]string{}
	// 物理端口对应接口
	r0, er0 := FetchItems("walk", node, []string{OLibrary.Dot1dBasePortIfIndex}, snmp)

	if er0 == nil {
		// mac对应物理端口
		r1, er1 := FetchItems("walk", node, []string{OLibrary.Dot1dTpFdbPort}, snmp)
		if er1 == nil {

			for _, v := range r0 {
				tmp := strings.Split(v.Oid, ".")
				p := tmp[len(tmp)-1]
				portIfIndex[p] = v.DataValue.(*big.Int).Int64()
				ifIndexPort[v.DataValue.(*big.Int).Int64()] = p
			}
			for _, v := range r1 {
				macIfIndex[strings.ReplaceAll(strings.TrimPrefix(v.Oid, OLibrary.Dot1dTpFdbPort+"."), ".", " ")] =
					portIfIndex[fmt.Sprintf("%v", v.DataValue)]
			}
		}
	}
	r2, er2 := FetchItems("walk", node, []string{OLibrary.IfDes}, snmp)
	if er2 == nil {
		for _, v := range r2 {
			index, er := strconv.ParseInt(strings.TrimPrefix(v.Oid, OLibrary.IfDes+"."), 10, 64)
			if er == nil {
				ifIndexDes[index] = fmt.Sprintf("%v", v.DataValue)
			}
		}
	}
	for index, port := range ifIndexPort {
		data[index] = &IfIndexInfo{
			IfIndex: index,
			Port:    port,
			Des:     ifIndexDes[index],
		}
	}
	for k, v := range macIfIndex {
		if _, ok := data[v]; ok {
			data[v].ToIps = append(data[v].ToIps, k)
		}
	}
}

func ifIndexMapIpRoute(node string, data map[int64]*IfIndexInfo, snmp *gsnmp.GoSNMP) {
	var (
		types      = map[string]int64{}
		nextHops   = map[string]string{}
		ifIndexDes = map[int64]string{}
	)
	r, er := FetchItems("walk", node, []string{OLibrary.IfDes}, snmp)
	if er == nil {
		for _, v := range r {
			index, er := strconv.ParseInt(strings.TrimPrefix(v.Oid, OLibrary.IfDes+"."), 10, 64)
			if er == nil {
				ifIndexDes[index] = fmt.Sprintf("%v", v.DataValue)
			}
		}
	}
	r0, er0 := FetchItems("walk", node, []string{OLibrary.IpRouteType}, snmp)
	if er0 == nil {
		r1, er1 := FetchItems("walk", node, []string{OLibrary.IpRouteNextHop}, snmp)
		if er1 == nil {
			r2, er2 := FetchItems("walk", node, []string{OLibrary.IpRouteIfIndex}, snmp)
			if er2 == nil {
				for _, v := range r0 {
					types[strings.TrimPrefix(v.Oid, OLibrary.IpRouteType)] = v.DataValue.(*big.Int).Int64()
				}
				for _, v := range r1 {
					nextHops[strings.TrimPrefix(v.Oid, OLibrary.IpRouteNextHop)] = v.DataValue.(string)
				}
				for _, v := range r2 {
					k := strings.TrimPrefix(v.Oid, OLibrary.IpRouteIfIndex)
					ifIndex := v.DataValue.(*big.Int).Int64()
					if v1, ok := types[k]; ok && v1 == 4 {
						if _, ok1 := data[ifIndex]; ok1 {
							if !StringInSlice(nextHops[k], data[ifIndex].ToIps) {
								data[ifIndex].ToIps = append(data[ifIndex].ToIps, nextHops[k])
							}
						} else {
							data[ifIndex] = &IfIndexInfo{
								IfIndex: ifIndex,
								ToIps:   []string{nextHops[k]},
								Port:    fmt.Sprintf("%v", ifIndex),
								Des:     ifIndexDes[ifIndex],
							}
						}
					}
				}
			}
		}

	}
}

func IfIndexMapIp(node string, snmp *gsnmp.GoSNMP) (ident string, data map[int64]*IfIndexInfo, err error) {
	data = map[int64]*IfIndexInfo{}
	if snmp == nil {
		snmp, er := NewSnmp(node)
		if er != nil {
			err = er
			return
		} else {
			defer snmp.Conn.Close()
		}
	}
	ident, _, err = IpIdentGet(node, snmp)
	if err != nil {
		return
	}
	if _, ok := IfIndexMapIpRecord.LoadOrStore(ident, struct{}{}); ok {
		return ident, data, nil
	}
	// 对于交换机
	ifIndexMapIpSwitch(ident, data, snmp)
	// 对于有路由的情况
	if len(data) == 0 {
		ifIndexMapIpRoute(ident, data, snmp)
	}
	for k := range data {
		data[k].ToIps = RmDupElement(data[k].ToIps)
		for _, v1 := range data[k].ToIps {
			if v2, ok := IpNodeTypeMaps.Get(v1); ok {
				switch v2.(NodeType) {
				case NTSwitch, NTRouter:
					if ip, _, er := IpIdentGet(v1, nil); er == nil && ip != "" && ip != node {
						data[k].Neighbor = ip
						goto Loop
					}
				}
			}
		}
	Loop:
		continue
	}
	return
}

// ScanArp 获取mac，ip的映射关系
func ScanArp(ipAddr string, snmp *gsnmp.GoSNMP) (macIpsMap map[string][]string, er error) {
	if v, ok := IpIdent.Get(ipAddr); ok {
		ipAddr = v.(string)
	}
	if _, ok := ScanArpFinished.LoadOrStore(ipAddr, false); ok {
		return
	}
	if snmp == nil {
		snmp, er = NewSnmp(ipAddr)
		if er != nil {
			return
		}
	}
	arpSnmp, err := FetchItems("walk", ipAddr, []string{OLibrary.IpNetToMediaPhysAddress}, snmp)
	if err != nil {
		er = err
		return
	}
	r1, err := FetchItems("walk", ipAddr, []string{OLibrary.Dot1dBaseBridgeAddress}, snmp)
	if err != nil {
		return
	}
	macIpsMap = make(map[string][]string)
	//macIpsMap = make(map[interface{}][]string)

	for _, v := range arpSnmp {
		tmp := strings.Split(v.Oid, ".")
		if len(tmp) < 5 {
			continue
		}
		//h := HardwareAddr(v.Raw)
		h := ToHardwareAddr(v.Raw)
		if h == "" {
			continue
		}
		macIpsMap[h] = append(macIpsMap[h], strings.Join(tmp[len(tmp)-4:], "."))
		//if _, ok := macIpsMap[h]; ok {
		//	macIpsMap[h] = append(macIpsMap[h], strings.Join(tmp[len(tmp)-4:], "."))
		//} else {
		//	macIpsMap[h] = []string{strings.Join(tmp[len(tmp)-4:], ".")}
		//}
	}
	for _, v := range r1 {
		//h := HardwareAddr(v.Raw)
		h := ToHardwareAddr(v.Raw)
		macIpsMap[h] = append(macIpsMap[h], ipAddr)
		//if _, ok := macIpsMap[h]; ok {
		//	macIpsMap[h] = append(macIpsMap[h], ipAddr)
		//} else {
		//	macIpsMap[h] = []string{ipAddr}
		//}
	}

	for k, v := range macIpsMap {
		MacIpsSet(k, v)
		for _, i := range v {
			//TotalIps.Store(i, struct{}{})
			IpMacAddress.Store(i, k)
		}
	}
	return
}

func MacIpsSet(k, v any) {
	commonLocker.Lock()
	defer commonLocker.Unlock()
	var tmp []string
	if v1, ok := MacIps.Load(k); ok {
		tmp = v1.([]string)
		tmp = append(tmp, v.([]string)...)
	} else {
		tmp = v.([]string)
	}
	MacIps.Store(k, RmDupElement(tmp))

}

type StpInfo struct {
	AgentIp       string
	AgentAddress  string
	Port          string
	PortState     int64
	DesignAddress string
	DesignPort    string
}

func NodeLink(node string, snmp *gsnmp.GoSNMP) {
	if _, ok := NodeLinkFinished.LoadOrStore(node, false); ok {
		return
	}

	r1, err := FetchItems("walk", node, []string{OLibrary.Dot1dBaseBridgeAddress}, snmp)
	if err != nil {
		return
	}
	r2, err := FetchItems("walk", node, []string{OLibrary.Dot1dStpPortDesignatedBridge}, snmp)
	if err != nil {
		return
	}
	r3, err := FetchItems("walk", node, []string{OLibrary.Dot1dStpPortState}, snmp)
	if err != nil {
		return
	}
	r4, err := FetchItems("walk", node, []string{OLibrary.Dot1dStpPort}, snmp)
	if err != nil {
		return
	}
	r5, err := FetchItems("walk", node, []string{OLibrary.Dot1dStpPortDesignatedPort}, snmp)
	if err != nil {
		return
	}

	selfMacAddress := ""
	for _, v := range r1 {
		//selfMacAddress = HardwareAddr(v.Raw)
		selfMacAddress = ToHardwareAddr(v.Raw)
	}
	var states = map[string]int64{}
	for _, v := range r3 {
		states[strings.TrimPrefix(v.Oid, OLibrary.Dot1dStpPortState)] = v.DataValue.(*big.Int).Int64()
	}
	var Ports = map[string]string{}
	for _, v := range r4 {
		Ports[strings.TrimPrefix(v.Oid, OLibrary.Dot1dStpPort)] = fmt.Sprintf("%v", v.Raw)
	}
	var dPorts = map[string]string{}
	for _, v := range r5 {
		dPorts[strings.TrimPrefix(v.Oid, OLibrary.Dot1dStpPortDesignatedPort)] = fmt.Sprintf("%v", v.Raw)
	}

	for _, v := range r2 {
		if ToHardwareAddr(v.Raw)[6:] == selfMacAddress {
			continue
		}
		index := strings.TrimPrefix(v.Oid, OLibrary.Dot1dStpPortDesignatedBridge)

		if v1, ok := states[strings.TrimPrefix(v.Oid, OLibrary.Dot1dStpPortDesignatedBridge)]; ok && v1 == 5 {
			data := StpInfo{
				AgentIp:       node,
				AgentAddress:  selfMacAddress,
				Port:          Ports[index],
				PortState:     states[index],
				DesignAddress: ToHardwareAddr(v.Raw)[6:],
				DesignPort:    dPorts[index],
			}
			NodeLinks.Store(selfMacAddress, data)
			//fmt.Println("store:::", node, v.Raw, HardwareAddr(v.Raw)[6:], selfMacAddress)
		}
		break
	}
}
