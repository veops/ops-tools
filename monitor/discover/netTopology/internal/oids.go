// Package netTopology /*
package internal

// Sys 设备信息, Get 信息
type Sys struct {
	// SysName 主机名
	SysName string
	// SysDesc 产商信息
	SysDesc string
	// SysUpTimeInstance 监控时间
	SysUpTimeInstance string
	// SysLocation 设备位置
	SysLocation string
}

// Dot1dBridge mac地址表,即mac与端口的映射关系
type Dot1dBridge struct {
	// Dot1dTpFdbAddress 桥接转发表, mac地址
	// 在这里，需要用到Community String Indexing，思科设备的转发表对每个vlan是不一样的。这时候共同体名为community@vlank号，
	// 如：public@1,public@2,public@3等,默认取public@1地的转发表。此时需在对每个vlan的转发表进行查找，直到找到ip对应的mac地址
	Dot1dTpFdbAddress string
	// Dot1dTpFdbPort 转发端口, 对应物理端口 Port Number
	Dot1dTpFdbPort string
}

type Dot1dStp struct {
	// Dot1dStpPortTable
	Dot1dStpPortTable string
	// Dot1dStpPortDesignatedBridge 对应端口所连接的另一台二层设备的BridgeID, 与dot1dBaseBridgeAddress对应
	Dot1dStpPortDesignatedBridge string
	// Dot1dStpPortDesignatedPort 该二层设备的某个端口连接了另一台二层设备的哪个桥接端口
	Dot1dStpPortDesignatedPort string
	// Dot1dStpRootCost 从该网桥到根的路径成本
	// 两台交换机之间的连接信息是由其值较大(离root较远)的那一台交换机产生的,另一台则以自身信息作为默认
	// 值越大，在生成树种的层级越低，两台交换机的连接关系由层级较低的交换机确定
	Dot1dStpRootCost string
	// Dot1dStpPort 二层代理设备各个端口的桥接端口号/包含生成树协议管理信息的端口的端口号
	Dot1dStpPort string
	// Dot1dStpPortState 表明端口的工作状态
	//1: disabled; 2:blocking(冗余链路); 3:listening; 4:learning; 5:forwarding; 6:broken
	// spanning-tree技术把冗余角色的备用线路设置成forwarding(5)以外的其他状态
	Dot1dStpPortState string
}

type Dot1dBase struct {
	// Dot1dBasePort 物理端口号
	Dot1dBasePort string
	// Dot1dBasePortIfIndex 端口对应的接口索引
	Dot1dBasePortIfIndex string
	// Dot1dBaseBridgeAddress 代表设备自身MAC地址 .1.3.6.1.2.1.17.1.1
	Dot1dBaseBridgeAddress string
}

// IpNetToMedia 该表是一张IP地址转换表，主要是将IP地址映射为物理地址;该表的索引是ipNetToMediaIfIndex和ipNetToMediaNetAddress
// 通过其可以发现与该设备相连的其他设备的ip地址和物理地址的映射以及映射方式，从而发现一些设备的连接情况
type IpNetToMedia struct {
	// ipNetToMediaIfIndex 表示此表项对应的有效接口的索引值, 该值所指定的接口与IF-MIB中的ifIndex值所指定接口相同
	IpNetToMediaIfIndex string
	// IpNetToMediaPhysAddress arp表. 表示依据媒介而定的物理地址, 即ip与mac地址的映射关系
	IpNetToMediaPhysAddress string
	// IpNetToMediaNetAddress arp表中 mac对应IP, 和本机在一个子网的ip。 表示这个依据媒介而定的物理地址对应的IP地址。
	IpNetToMediaNetAddress string
	// ipNetToMediaType other(1),invalid(2) dynamic(3):动态，即对端设备的ip;static(4):静态,代表本机的Ip
	IpNetToMediaType string
}

type Ip struct {
	//ipForwarding = ""
	// IpForwarding 是否具有路由功能， 1: 有路由功能 2: 无路由功能
	IpForwarding string
	// PrintMib 是否为打印机，如不为空，则为打印机
	PrintMib string
}

type IpRoute struct {
	// IpRouteDest 路由表(目标网络)
	IpRouteDest string
	// IpRouteIfIndex 路由表（出接口索引 唯一标识本地接口的索引值,通过该接口可以到达该路由的下一站）
	IpRouteIfIndex string
	// IpRouteNextHop 显示这条路由下一跳的IP地址. 当路由与广播媒介接口绑定时，该节点的值为接口上代理的IP地址
	IpRouteNextHop string
	// IpRouteType路由类型
	// 3(direct):直接路由,表明目标网络或者目标主机与该路由器直接相连
	// 4(indirect):间接路由,表明通往目的网络或者目的主机的的路径上还要经过其他路由器
	// 直连路由：直接连接到路由器端口的网段，该信息由路由器自动生成；
	// 非直连路由：不是直接连接到路由器端口的网段，此记录需要手动添加或使用动态路由生成。
	IpRouteType string
	IpRouteMask string
}

type If struct {
	// IfName interface 名称
	IfName  string
	IfIndex string
	// 网卡描述
	IfDes string
	// IfHighSpeed 带宽限制
	IfHighSpeed   string
	IfHCInOctets  string
	IfHCOutOctets string
	IfInDiscards  string
	IfOutDiscards string

	// IfOutUcastPkts 输出非广播包数
	IfOutUcastPkts string
	IfInUcastPkts  string
	// ifOutNUcastPkts 广播包和多点发送包计数，速率可以识别广播风暴
	IfOutNUcastPkts string
	IfInNUcastPkts  string
	IfOutErrors     string
	IfInErrors      string
	// IfAdminStatus 接口的配置状态; up(1),down(2),testing(3)
	IfAdminStatus string
	// IfOperStatus 接口的当前工作状态up(1),down(2),testing(3)
	IfOperStatus string
}

// IpAdEnt ipAddrTable 该表主要是用来保存IP地址信息，如IP地址、子网掩码等; 该表的索引是ipAdEntAddr。
type IpAdEnt struct {
	// IpAdEntAddr 显示这个表项的地址信息所属的IP地址
	IpAdEntAddr string
	// IpAdEntIfIndex 唯一标识该表项所应用的接口的索引值
	IpAdEntIfIndex string
	// IpAdEntNetMask 显示该IP地址的子网掩码
	IpAdEntNetMask string
}

type EntPhys struct {
	// entPhysicalDescr 设备各模块描述
	EntPhysicalDescr string
	// EntPhysicalName 设备模块物理名称
	EntPhysicalName string
	// EntPhysicalSerialNum 设备各模块序列号
	EntPhysicalSerialNum string
	// EntPhysicalMfgName 设备各模块产商
	EntPhysicalMfgName string
}

type Neighbor struct {
	// Neighbor  LLDP邻居信息 1.0.8802.1.1.2.1.4
	Neighbor string
}

type AtEntry struct {
	AtPhysAddress string
}

type OidLibrary struct {
	Sys
	Dot1dBridge
	Dot1dBase
	IpNetToMedia
	Ip
	IpRoute
	If
	IpAdEnt
	Dot1dStp
	EntPhys
}

var (
	OLibrary = &OidLibrary{
		Sys{
			SysName:           ".1.3.6.1.2.1.1.5.",
			SysDesc:           ".1.3.6.1.2.1.1.1",
			SysUpTimeInstance: ".1.3.6.1.2.1.1.3",
			SysLocation:       ".1.3.6.1.2.1.1.6",
		},
		Dot1dBridge{
			Dot1dTpFdbAddress: ".1.3.6.1.2.1.17.4.3.1.1",
			Dot1dTpFdbPort:    ".1.3.6.1.2.1.17.4.3.1.2",
		},
		Dot1dBase{
			Dot1dBaseBridgeAddress: ".1.3.6.1.2.1.17.1.1.0",
			Dot1dBasePort:          ".1.3.6.1.2.1.17.1.4.1.1",
			Dot1dBasePortIfIndex:   ".1.3.6.1.2.1.17.1.4.1.2",
		},
		IpNetToMedia{
			IpNetToMediaIfIndex:     ".1.3.6.1.2.1.4.22.1.1",
			IpNetToMediaPhysAddress: ".1.3.6.1.2.1.4.22.1.2",
			IpNetToMediaNetAddress:  ".1.3.6.1.2.1.4.22.1.3",
			IpNetToMediaType:        ".1.3.6.1.2.1.4.22.1.4",
		},
		Ip{
			IpForwarding: ".1.3.6.1.2.1.4.1",
			PrintMib:     ".1.3.6.1.2.1.43",
		},
		IpRoute{
			IpRouteDest:    ".1.3.6.1.2.1.4.21.1.1",
			IpRouteIfIndex: ".1.3.6.1.2.1.4.21.1.2",
			IpRouteNextHop: ".1.3.6.1.2.1.4.21.1.7",
			IpRouteType:    ".1.3.6.1.2.1.4.21.1.8",
			IpRouteMask:    ".1.3.6.1.2.1.4.21.1.11",
		},
		If{
			IfName:          ".1.3.6.1.2.1.31.1.1.1.1",
			IfIndex:         ".1.3.6.1.2.1.2.2.1.1",
			IfDes:           ".1.3.6.1.2.1.2.2.1.2",
			IfAdminStatus:   ".1.3.6.1.2.1.2.2.1.7",
			IfOperStatus:    ".1.3.6.1.2.1.2.2.1.8",
			IfInUcastPkts:   ".1.3.6.1.2.1.2.2.1.11",
			IfInNUcastPkts:  ".1.3.6.1.2.1.2.2.1.12",
			IfInDiscards:    ".1.3.6.1.2.1.2.2.1.13",
			IfInErrors:      ".1.3.6.1.2.1.2.2.1.14",
			IfOutUcastPkts:  ".1.3.6.1.2.1.2.2.1.17",
			IfOutNUcastPkts: ".1.3.6.1.2.1.2.2.1.18",
			IfOutDiscards:   ".1.3.6.1.2.1.2.2.1.19",
			IfOutErrors:     ".1.3.6.1.2.1.2.2.1.20",

			IfHighSpeed:   ".1.3.6.1.2.1.31.1.1.1.15",
			IfHCInOctets:  ".1.3.6.1.2.1.31.1.1.1.6",
			IfHCOutOctets: ".1.3.6.1.2.1.31.1.1.1.10",

			//ifAdminStatus
			//
			//
			//
			//用于配置接口的状态（可读写）up(1),down(2),testing(3)（见表2）
			//
			//ifOperStatus
			//
			//
			//
			//提供接口的当前工作状态up(1),down(2),testing(3)
			//
			//（见表2）
		},
		IpAdEnt{
			IpAdEntAddr:    ".1.3.6.1.2.1.4.20.1.1",
			IpAdEntIfIndex: ".1.3.6.1.2.1.4.20.1.2",
			IpAdEntNetMask: ".1.3.6.1.2.1.4.20.1.3",
		},
		Dot1dStp{
			Dot1dStpRootCost:             ".1.3.6.1.2.1.17.2.6",
			Dot1dStpPortTable:            ".1.3.6.1.2.1.17.2.15",
			Dot1dStpPort:                 ".1.3.6.1.2.1.17.2.15.1.1",
			Dot1dStpPortState:            ".1.3.6.1.2.1.17.2.15.1.3",
			Dot1dStpPortDesignatedBridge: ".1.3.6.1.2.1.17.2.15.1.8",
			Dot1dStpPortDesignatedPort:   ".1.3.6.1.2.1.17.2.15.1.9",
		},
		EntPhys{
			EntPhysicalDescr:     ".1.3.6.1.2.1.47.1.1.1.1.2",
			EntPhysicalName:      ".1.3.6.1.2.1.47.1.1.1.1.7",
			EntPhysicalSerialNum: ".1.3.6.1.2.1.47.1.1.1.1.11",
			EntPhysicalMfgName:   ".1.3.6.1.2.1.47.1.1.1.1.12",
		},
	}
)
