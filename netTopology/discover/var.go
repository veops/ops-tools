package discover

import (
	"sync"
	"time"

	"github.com/patrickmn/go-cache"

	. "netTopology/internal"
)

//type Node struct {
//	// Name must be unique
//	Name string `json:"name"`
//	// Alias
//	Alias string `json:"alias"`
//	// Type switch/router/server/internet/firewall/unknown
//	Type string `json:"type"`
//	// X location x
//	X int `json:"x"`
//	// Y location y
//	Y int `json:"y"`
//	// Ips 本机IP
//	Ips []string `json:"ips"`
//	// Locals 直连ip
//	Locals []string `json:"locals"`
//	// discoverIps 间接连接ip
//	discoverIps []string
//	Options     interface{} `json:"options"`
//	Status      string      `json:"status"`
//}

//  name2: Router_symbol_(96)
//  #bandwidth: 10               # 可以覆盖默认的link里的设置
//  #width: 5
//  #copy: true                  # 如果设置，即使拓扑里没这个元素，也会画出来
//  hostname: Router             # 设置监控项采集的节点
//  itemin: net.if.in["eth0"]    # 设置监控项的key
//  itemout: net.if.out["eth0"]  # 设置监控项的key

type Link struct {
	NodeFrom     string  `json:"node_from"`
	NodeTo       string  `json:"node_to"`
	NodeFromName string  `json:"node_from_name"`
	NodeToName   string  `json:"node_to_name"`
	Value        float64 `json:"value"`
	Unit         string  `json:"unit"`
	Level        uint8   `json:"level"`
	IfIndex      string  `json:"if_index"`
	TrafficLoad  float64 `json:"traffic_load"`
	// Port 物理接口
	Port    string      `json:"port"`
	Options interface{} `json:"options"`
}

type TopologyGraph struct {
	Name     string  `json:"name"`
	Nodes    []*Node `json:"nodes"`
	Links    []*Link `json:"links"`
	Location bool    `json:"location"`
	GraphId  string  `json:"graph_id"`
}

type Arp struct {
	Target string
	// SelfIps 自身IP
	SelfIps []string
	// Subnets 连接的子网
	Subnets []string
	// LocalIps 子网ip
	LocalIps []string
	// PeerIps 间接ip列表,即平行连接ip列表，如交换机的nextHop
	PeerIps []string
}

const ()

var (
	topology3Lock = sync.RWMutex{}

	IpScanWaiting = make(chan string, 10000)
	// IpScannedRecord 记录Ip是否扫描
	IpScannedRecord = sync.Map{}
	// IpEntOfDevice 设备的包含的IP列表
	IpEntOfDevice     = sync.Map{}
	Topology3SyncWait = sync.WaitGroup{}
	// NetDevices 存放发现的
	NetDevices       = make(chan string, 10000)
	DiscoverIpsCache = cache.New(time.Minute*5, time.Minute)

	//TargetStatus     = cache.New(time.Hour, time.Minute*30)
	//CommunityMap     = cache.New(time.Hour*24, time.Hour)

	MacIps       = sync.Map{}
	L3Nodes      = sync.Map{}
	L3Links      = sync.Map{}
	Routes       = sync.Map{}
	L2Switches   = sync.Map{}
	IpMacAddress = sync.Map{}
	L2IpScanned  = sync.Map{}

	SubnetsVar = &Subnets{M: map[string]*Subnet{}}
	// NodeLinkFinished 是否完成NodeLink
	NodeLinkFinished = sync.Map{}
	ScanArpFinished  = sync.Map{}
	SubnetsScanned   = sync.Map{}
	TotalIps         = sync.Map{}
	LocalMetrics     = make(chan *Metric, 1000)
	// TG map[string]*TopologyGraph{}
	TG = sync.Map{}
)

func Init() {
	IpScanWaiting = make(chan string, 10000)
	IpScannedRecord = sync.Map{}
	IpEntOfDevice = sync.Map{}
	Topology3SyncWait = sync.WaitGroup{}
	NetDevices = make(chan string, 10000)
	DiscoverIpsCache = cache.New(time.Minute*5, time.Minute)
	MacIps = sync.Map{}
	L3Nodes = sync.Map{}
	L3Links = sync.Map{}
	Routes = sync.Map{}
	IpMacAddress = sync.Map{}
	NodeLinkFinished = sync.Map{}
	ScanArpFinished = sync.Map{}
	SubnetsScanned = sync.Map{}
	TotalIps = sync.Map{}

	ResultCache = cache.New(time.Hour, time.Minute*15)
	TargetStatus = cache.New(time.Hour, time.Minute*30)
	//CommunityMap = cache.New(time.Hour*24, time.Hour)
	//IpIdent = sync.Map{}
	IpNodeTypeMaps = cache.New(time.Hour*24*7, time.Hour)
}
