package internal

import (
	"fmt"
	"github.com/go-ping/ping"
	gsnmp "github.com/gosnmp/gosnmp"
	"math/big"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// CidrSpan 默认最大网络号
	CidrSpan = 20
	// SpecialCIDRs 对于超过4094位的需要单独指定
	// 如 {"1.1.1.1/19":""}
	SpecialCIDRs = sync.Map{}
)

func InetItoa(ip int64) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

func InetAtoi(ip string) int64 {
	ret := big.NewInt(0)
	ret.SetBytes(net.ParseIP(strings.TrimSpace(ip)).To4())
	return ret.Int64()
}

func PingHost(hostName string) bool {
	p, err := ping.NewPinger(hostName)
	if err != nil {
		return false
	}
	p.Timeout = time.Second
	p.SetPrivileged(true)
	p.Count = 3
	err = p.Run() // Blocks until finished.
	if err != nil {
		return false
	}
	return p.Statistics().PacketsRecv > 0
}

// PortScan 扫描ip端口
// @params ignorePort bool 是否忽略端口返回,此种情况只是为了在icmp协议禁用之后,判断主机是否在线
// @return available bool 是否在线
// @return ports []int 在线端口列表
func PortScan(ipStr string, ignorePort bool, scanPorts []int) (available bool, ports []int) {
	var wg sync.WaitGroup
	var ch = make(chan struct{}, 50)
	var lock sync.Mutex
	var chClosed atomic.Bool
	chClosed.Store(false)
	for j := 1; j <= 65535; j++ {
		if len(scanPorts) > 0 && !InSlice(j, scanPorts) {
			continue
		}
		wg.Add(1)
		if chClosed.Load() {
			wg.Done()
			break
		}
		ch <- struct{}{}
		go func(i int) {
			defer wg.Done()
			<-ch
			if chClosed.Load() {
				return
			}
			var address = fmt.Sprintf("%s:%d", ipStr, i)
			conn, err := net.DialTimeout("tcp", address, time.Second*10)
			if err != nil {
				return
			}
			conn.Close()
			if ignorePort {
				chClosed.Store(true)
			} else {
				lock.Lock()
				ports = append(ports, i)
				lock.Unlock()
			}
		}(j)
	}
	wg.Wait()
	return
}

func InSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func StringConcat(str ...string) string {
	var builder strings.Builder
	for _, v := range str {
		builder.WriteString(v)
	}
	return builder.String()
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func containsCIDR(cr1, cr2 string) bool {
	_, a, _ := net.ParseCIDR(cr1)
	_, b, _ := net.ParseCIDR(cr2)
	ones1, _ := a.Mask.Size()
	ones2, _ := b.Mask.Size()
	return ones1 <= ones2 && a.Contains(b.IP)
}

// acceptCidr 当获取一个网段时,网段太大，扫描负担很重，因此默认值采集最小255.255.16.0，即网络号至少20位是网络号
// 如有特殊需求，需明确指定cidr范围
// cidr: 192.168.1.1/24
func validtCidr(cidr string) bool {
	//tmp := strings.Split(cidr, "/")
	//var sz int
	//// 1.1.1.1/255.255.255.0
	//if len(tmp) == 2 && len(tmp[1]) > 2 {
	//	adr := net.ParseIP(tmp[1]).To4()
	//	sz, _ = net.IPv4Mask(adr[0], adr[1], adr[2], adr[3]).Size()
	//	if sz >= CidrSpan {
	//		return true
	//	} else {
	//		cidr = fmt.Sprintf("%s/%d", tmp[0], sz)
	//	}
	//}
	_, ipNet, _ := net.ParseCIDR(cidr)
	if m, _ := ipNet.Mask.Size(); m >= CidrSpan {
		return true
	}
	var valid bool
	SpecialCIDRs.Range(func(key, value any) bool {
		if containsCIDR(key.(string), cidr) {
			valid = true
			return false
		}
		return true
	})
	return valid
}

// CidrHosts  Convert Cidr Address To Hosts
// newAddr : "192.168.1.0/24" or "192.168.1.0/255.255.255.0"
// 防止返回过多ip,此处限制最多返回一个B段
func CidrHosts(netAddr string) ([]string, error) {
	tmp := strings.Split(netAddr, "/")
	if len(tmp) == 2 && len(tmp[1]) > 2 {
		adr := net.ParseIP(tmp[1]).To4()
		sz, _ := net.IPv4Mask(adr[0], adr[1], adr[2], adr[3]).Size()
		netAddr = fmt.Sprintf("%s/%d", tmp[0], sz)
	}
	if !validtCidr(netAddr) {
		return nil, fmt.Errorf("ip range is too large for %s, skipped", netAddr)
	}
	ipAddr, ipv4Net, err := net.ParseCIDR(netAddr)
	if err != nil {
		return nil, err
	}
	var ips []string
	for ip := ipAddr.Mask(ipv4Net.Mask); ipv4Net.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
		if len(ips) > 65535 {
			break
		}
	}
	//mask := binary.BigEndian.Uint32(ipv4Net.Mask)
	//start := binary.BigEndian.Uint32(ipv4Net.IP)
	//finish := (start & mask) | (mask ^ 0xffffffff)
	//var hosts []string
	//tmp1 := 0
	//for i := start + 1; i <= finish-1; i++ {
	//	if tmp1 > 100 {
	//		fmt.Println("break", finish, mask, start, mask, start&mask, mask^0xffffffff)
	//		break
	//	}
	//	ip := make(net.IP, 4)
	//	binary.BigEndian.PutUint32(ip, i)
	//	hosts = append(hosts, ip.String())
	//	tmp1++
	//}
	if len(ips) > 2 {
		return ips[1 : len(ips)-1], nil
	}
	return []string{}, nil
}

// MacAddressConvert
// from: 11.21.31.41.51.61 --> b:15:1f:29:33:3d
func MacAddressConvert(from string) string {
	//.1.3.6.1.2.1.4.22.1.2
	v := strings.Split(from, ".")
	var s []string
	for _, v1 := range v {
		if v1 == "" {
			continue
		}
		d, er := strconv.Atoi(v1)
		if er != nil {
			return from
		}
		s = append(s, fmt.Sprintf("%x", d))
	}
	return strings.Join(s, ":")
}

func ValidInnerIp(ip string) bool {
	//
	validInnerIps := []string{
		// 10.0.0.0/8
		"10.0.0.0-10.255.255.255",
		// 172.16.0.0/12
		"172.16.0.0-172.31.255.255",
		// 192.168.0.0/16
		"192.168.0.0-192.168.255.255",
		// 198.18.0.0/15 用于测试两个独立子网的网间通信
		"198.18.0.0-198.19.255.255",
		// 198.51.100.0/24
		"198.51.100.0-198.51.100.255",
		//203.0.113.0/24
		"203.0.113.0-203.0.113.255",
	}
	for _, v := range validInnerIps {
		ipSlice := strings.Split(v, `-`)
		if len(ipSlice) < 0 {
			continue
		}
		if InetAtoi(ip) >= InetAtoi(ipSlice[0]) && InetAtoi(ip) <= InetAtoi(ipSlice[1]) {
			return true
		}
	}
	var valid bool
	SpecialCIDRs.Range(func(key, value any) bool {
		t := net.ParseIP(ip)
		if t != nil {
			_, a, _ := net.ParseCIDR(key.(string))
			if a.Contains(t) {
				valid = true
				return false
			}
		}
		return true
	})
	return valid
}

func HardwareAddr(addr interface{}) string {
	switch address := addr.(type) {
	case string:
		return address
	case []byte:
		return net.HardwareAddr(address).String()
	}
	return ""
}

func ToHardwareAddr(val interface{}) string {
	switch value := val.(type) {
	case string:
		tmp := strings.Split(value, " ")
		if len(tmp) == 6 {
			n := []string{}
			for _, v := range tmp {
				v1, er := strconv.ParseUint(v, 10, 64)
				if er != nil {
					return val.(string)
				}
				n = append(n, fmt.Sprintf("%x", v1))
			}
			return strings.Join(n, ":")
		} else {
			return val.(string)
		}
	case []byte:
		n := []string{}
		for _, v := range value {
			n = append(n, fmt.Sprintf("%x", v))
		}
		return strings.Join(n, ":")
	}
	return ""

}

func RmDupElement(arg []string) []string {
	tmp := map[string]int{}
	for _, el := range arg {
		if _, ok := tmp[el]; !ok {
			tmp[el] = 1
		}
	}
	var result []string
	for k := range tmp {
		result = append(result, k)
	}
	return result
}

// Final
// {"a":"b", "b":"c", "c":"d"}, final("a") wil get "d"
func Final(key string, data map[string]string) string {
	if v, ok := data[key]; ok {
		if key == v {
			return key
		}
		return Final(v, data)
	} else {
		return key
	}
}

//func macAddress(from string) string {
//	var tmp []string
//	for _, v1 := range strings.Split(from, " ") {
//		t, er := strconv.ParseUint(v1, 10, 8)
//		if er != nil {
//			break
//		}
//		tmp = append(tmp, fmt.Sprintf("%x", t))
//	}
//	if len(tmp) < 6 {
//		return from
//	}
//	tmp = tmp[len(tmp)-6:]
//	return strings.Join(tmp, ":")
//}

//func SnmpWalk(snmp *gsnmp.GoSNMP, action string, oids ...string) (res []*SnmpResp, err error) {
//	if len(oids) == 0 {
//		return res, fmt.Errorf("oid is none")
//	}
//	var result []gsnmp.SnmpPDU
//	if action == "get" {
//		r, er := snmp.Get(oids)
//		if er != nil {
//			err = er
//			return
//		}
//		result = r.Variables
//	} else {
//		result, err = snmp.BulkWalkAll(oids[0])
//		if err != nil {
//			return
//		}
//	}
//	for _, v := range result {
//		tmp := SnmpResp{}
//		tmp.Oid = v.Name
//		switch v.Type {
//		case gsnmp.IPAddress:
//			tmp.DataType = "string"
//			tmp.DataValue = fmt.Sprintf("%v", v.Value)
//		case gsnmp.OctetString:
//			tmp.DataType = "string"
//			value := ""
//			// HEX-STRING
//			if strings.Contains(strconv.Quote(string(v.Value.([]byte))), "\\x") {
//				for i := 0; i < len(v.Value.([]byte)); i++ {
//					value += fmt.Sprintf("%v", v.Value.([]byte)[i])
//					if i != (len(v.Value.([]byte)) - 1) {
//						value += " "
//					}
//				}
//				tmp.DataValue = value
//			} else {
//				value = string(v.Value.([]byte))
//				tmp.DataValue = value
//			}
//		default:
//			tmp.DataType = "integer"
//			tmp.DataValue = gsnmp.ToBigInt(v.Value)
//			//tmp.DataValue = fmt.Sprintf("%d", v.Value)
//		}
//		tmp.Raw = v.Value
//		res = append(res, &tmp)
//	}
//	return res, nil
//
//}

func SnmpWalk(snmp *gsnmp.GoSNMP, action string, oids ...string) (res []*SnmpResp, err error) {
	if len(oids) == 0 {
		return res, fmt.Errorf("oid is none")
	}
	var result []gsnmp.SnmpPDU
	if action == "get" {
		r, er := snmp.Get(oids)
		if er != nil {
			err = er
			return
		}
		result = r.Variables
	} else if action == "first" {
		err = snmp.BulkWalk(oids[0], func(dataUnit gsnmp.SnmpPDU) error {
			if dataUnit.Value != nil {
				result = append(result, dataUnit)
				return fmt.Errorf("%s", "skip")
			}
			return nil
		})
		if err != nil {
			return
		}
	} else if action == "firstNotNone" {
		err = snmp.BulkWalk(oids[0], func(dataUnit gsnmp.SnmpPDU) error {
			if dataUnit.Value != nil && dataUnit.Type == gsnmp.OctetString && len(dataUnit.Value.([]byte)) != 0 {
				result = append(result, dataUnit)
				return fmt.Errorf("%s", "skip")
			}
			return nil
		})
		if err != nil {
			return
		}
	} else {
		result, err = snmp.BulkWalkAll(oids[0])
		if err != nil {
			return
		}
	}
	for _, v := range result {
		tmp := parseResult(v)
		tmp.Raw = v.Value
		res = append(res, &tmp)
	}
	return res, nil
}

func parseResult(v gsnmp.SnmpPDU) SnmpResp {
	tmp := SnmpResp{}
	tmp.Oid = v.Name
	switch v.Type {
	case gsnmp.IPAddress, gsnmp.ObjectIdentifier:
		tmp.DataType = "string"
		tmp.DataValue = fmt.Sprintf("%v", v.Value)
	case gsnmp.OctetString:
		tmp.DataType = "string"
		value := ""
		// HEX-STRING
		if strings.Contains(strconv.Quote(string(v.Value.([]byte))), "\\x") {
			for i := 0; i < len(v.Value.([]byte)); i++ {
				value += fmt.Sprintf("%v", v.Value.([]byte)[i])
				if i != (len(v.Value.([]byte)) - 1) {
					value += " "
				}
			}
			tmp.DataValue = value
		} else {
			value = string(v.Value.([]byte))
			tmp.DataValue = value
		}
	default:
		tmp.DataType = "integer"
		tmp.DataValue = gsnmp.ToBigInt(v.Value)
		//tmp.DataValue = fmt.Sprintf("%d", v.Value)
	}
	return tmp
}
