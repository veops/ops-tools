// Package discover /*
package discover

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	gsnmp "github.com/gosnmp/gosnmp"
	"github.com/patrickmn/go-cache"
	. "netTopology/internal"
)

type PortDetail struct {
	Port  string
	Index string
	Name  string
}

type IfCollectorConfig struct {
	Oid   string
	Alias string
	Order int
	Group string `json:"-"`
}

// PortsDetail map[string]map[string]PortDetail
// map[ip]map[port]PortDetail
var (
	PortsDetail  = cache.New(time.Minute, time.Minute)
	IfCollectors = map[string]IfCollectorConfig{
		"net_if_in_octets":  {OLibrary.IfHCInOctets, "输入带宽", 1, "带宽"},
		"net_if_out_octets": {OLibrary.IfHCOutOctets, "输出带宽", 2, "带宽"},
		//"net_if_in_ucast_pkts":   {OLibrary.IfInUcastPkts, "输入非广播包数", 3, "非广播包数"},
		//"net_if_out_ucast_pkts":  {OLibrary.IfOutUcastPkts, "输出非广播包数", 4, "非广播包数"},
		"net_if_in_nucast_pkts":  {OLibrary.IfInNUcastPkts, "输入广播包数", 5, "广播包数"},
		"net_if_out_nucast_pkts": {OLibrary.IfOutNUcastPkts, "输出广播包数", 6, "广播包数"},
		"net_if_in_discards":     {OLibrary.IfInDiscards, "输入包丢弃数", 7, "包丢弃数"},
		"net_if_out_discards":    {OLibrary.IfOutDiscards, "输出包丢弃数", 8, "包丢弃数"},
		"net_if_in_errors":       {OLibrary.IfInErrors, "输入包错误数", 9, "包错误数"},
		"net_if_out_errors":      {OLibrary.IfOutErrors, "输出包错误数", 10, "包错误数"},
		"net_if_highspeed":       {OLibrary.IfHighSpeed, "最大带宽", 11, "最大带宽"},
		"net_if_oper_status":     {OLibrary.IfOperStatus, "接口配置状态", 12, "接口配置状态"},
		"net_if_admin_status":    {OLibrary.IfOperStatus, "接口工作状态", 12, "接口工作状态"},
		//ifAdminStatus是指配置状态。
		//ifOperStatus是指实际工作状态
	}
)

func GetIfDetails(target string, snmpClient *gsnmp.GoSNMP) (data map[string]PortDetail, err error) {
	if v, ok := PortsDetail.Get(target); ok {
		data = v.(map[string]PortDetail)
		return
	}
	data = map[string]PortDetail{}
	if snmpClient == nil {
		snmpClient, err = NewSnmp(target)
		if err != nil {
			err = fmt.Errorf("new snmp client for %s failed %s", target, err.Error())
			return
		} else {
			defer snmpClient.Conn.Close()
		}
	}

	res1, er := FetchItems("walk", target, []string{OLibrary.IfDes}, snmpClient)
	if er != nil {
		err = errors.New(StringConcat("fetch item failed for ", target, OLibrary.IfDes))
		return
	}
	for _, v := range res1 {
		port := strings.TrimPrefix(v.Oid, fmt.Sprintf("%s.", OLibrary.IfDes))
		data[port] = PortDetail{Name: v.DataValue.(string), Port: port}
	}
	res2, er := FetchItems("walk", target, []string{OLibrary.IfIndex}, snmpClient)
	if er != nil {
		err = errors.New(StringConcat("fetch item failed for ", target, OLibrary.IfDes))
		return
	}
	for _, v := range res2 {
		port := strings.TrimPrefix(v.Oid, fmt.Sprintf("%s.", OLibrary.IfIndex))
		if v1, ok := data[port]; ok {
			v1.Index = fmt.Sprintf("%s", v.DataValue)
			data[port] = v1
		}
	}
	PortsDetail.SetDefault(target, data)
	return
}

func ifStatistics(target string) (data map[string]map[string]int64, err error) {
	snmpClient, er := NewSnmp(target)
	if er != nil {
		e := fmt.Errorf("new snmp client for %s failed %s", target, er.Error())
		return nil, e
	} else {
		defer snmpClient.Conn.Close()
	}

	data = map[string]map[string]int64{}
	port := ""
	for k, v := range IfCollectors {
		res, er := FetchItems("walk", target, []string{v.Oid}, snmpClient, 2)
		if er != nil {
			err = er
			return
		}
		data[k] = map[string]int64{}
		for _, v1 := range res {
			port = strings.TrimPrefix(v1.Oid, fmt.Sprintf("%s.", v.Oid))
			data[k][port] = v1.DataValue.(*big.Int).Int64()
		}
	}
	return data, nil
}
