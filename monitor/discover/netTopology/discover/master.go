// Package discover /*
package discover

import (
	"github.com/sirupsen/logrus"
	"math"
	"sort"
	"time"

	"github.com/jackpal/gateway"
	"github.com/shopspring/decimal"

	. "netTopology/internal"
)

type MasterDiscover string

func init() {
	go consumerMetrics()
}

// Collectors agents for collect and community config
func (h *MasterDiscover) Collectors(ident string) (data []*CollectorConfig, er error) {
	var nodes []*NodeSnmp
	nodes = append(nodes, &NodeSnmp{"", "public", "2c"})
	data = append(data, &CollectorConfig{
		Id:      "id1",
		Gateway: "",
		Nodes:   nodes,
		Name:    "test",
	})
	return data, nil
	//var resp []*GraphSnmp
	//q := map[string]interface{}{"enable": true}
	//if ident != "" {
	//	q["one_agent"] = ident
	//}
	//_, er = (&GraphSnmp{}).List(db.SnmpTopologyCol, q, int64(1), int64(1000), int64(1000), &resp)
	//if er != nil {
	//	return
	//}
	//for _, v := range resp {
	//	var nodes []*NodeSnmp
	//	for _, v1 := range v.Nodes {
	//		nodes = append(nodes, &NodeSnmp{v1.Ip, v1.Community, v1.Version})
	//	}
	//	data = append(data, &CollectorConfig{
	//		Id:      v.Id.Hex(),
	//		Gateway: v.Gateway,
	//		Nodes:   nodes,
	//		Name:    v.Name,
	//	})
	//}
	//return
}

// UploadRecord save graph on disk
func (h *MasterDiscover) UploadRecord(graph *TopologyGraph) error {
	//r, er := json.Marshal(graph)
	//if er != nil {
	//	return er
	//} else {
	//	_, err := utils.RedisAction("HSET", RedisKeyDiscoverLinks, graph.GraphId, r)
	//	if err != nil {
	//		return err
	//	}
	//}
	////
	//updateLinks(graph.Links, graph.GraphId)
	TG.Store(graph.GraphId, graph)
	return nil
}

func (h *MasterDiscover) Topology(config *CollectorConfig) (*TopologyGraph, error) {
	Init()
	graph := &TopologyGraph{GraphId: config.Id}

	if config.Gateway == "" {
		myGateway, er := gateway.DiscoverGateway()
		if er != nil {
			return graph, nil
		}
		config.Gateway = myGateway.String()
	}

	Topology3SyncWait.Add(1)
	for _, v := range config.Nodes {
		CommunityMap.SetDefault(v.Ip, v.Community)
		SnmpVersionMap.SetDefault(v.Ip, v.Version)
	}
	graph.Name = config.Name
	L3Topology(config.Gateway, graph, true)
	Topology3SyncWait.Wait()
	portMapIfIndex, portMapDes := ScanTotalSubnetsAndNeighbor(graph)
	tidyGraph(graph, portMapIfIndex, portMapDes)
	return graph, nil
}

func (h *MasterDiscover) Nodes() []string {
	//var (
	//	res  []string
	//	res1 = map[string]string{}
	//	res2 = map[string]string{}
	//)
	//err := utils.ScanMap(RedisKeyCompleteLinks, res1)
	//if err != nil {
	//	return res
	//}
	//nodes := map[string]struct{}{}
	//for _, v := range res1 {
	//	data1 := TopologyGraphV1{}
	//	er := json.Unmarshal([]byte(v), &data1)
	//	if er != nil {
	//		_ = level.Warn(g.Logger).Log("module", "net", "msg", er.Error())
	//		continue
	//	}
	//	for _, v1 := range data1.Nodes {
	//		nodes[v1.Name] = struct{}{}
	//	}
	//}
	//err = utils.ScanMap(RedisKeyDiscoverLinks, res2)
	//if err != nil {
	//	return res
	//}
	//for _, v := range res2 {
	//	data2 := TopologyGraph{}
	//	er := json.Unmarshal([]byte(v), &data2)
	//	if er != nil {
	//		_ = level.Warn(g.Logger).Log("module", "net", "msg", er.Error())
	//		continue
	//	}
	//	for _, v2 := range data2.Nodes {
	//		nodes[v2.Name] = struct{}{}
	//	}
	//}

	// -------simple-----
	nodes := map[string]struct{}{}
	TG.Range(func(key, value any) bool {
		for _, v := range value.(*TopologyGraph).Nodes {
			nodes[v.Name] = struct{}{}
		}
		return true
	})
	var res []string
	for k := range nodes {
		res = append(res, k)
	}
	sort.Strings(res)
	return res
}

func (h *MasterDiscover) UploadTsRecord(target string, data map[string]map[string]int64) {
	//var metrics []*pb.Metric
	ifDetail, er := GetIfDetails(target, nil)
	if er != nil {
		logrus.Error(er)
		return
	}
	var value float64
	t := time.Now().Unix()

	for metric, v := range data {
		for port, val := range v {
			if !math.IsInf(float64(val), 0) && !math.IsNaN(float64(val)) {
				value, _ = decimal.NewFromFloat(float64(val)).Round(2).Float64()
			} else {
				//fmt.Printf("invalid value %v for [%s  %s  %s]\n", val, node, valueType, direction)
				value = 0
			}
			if v2, ok := ifDetail[port]; ok {
				LocalMetrics <- &Metric{
					Metric: metric,
					Type:   "counter",
					Tags: map[string]string{
						"host": target,
						"if":   v2.Index,
						"port": v2.Port,
						"name": v2.Name,
					},
					Timestamp: t,
					Value:     value,
				}
			}
		}
	}
}

func consumerMetrics() {
	for m := range LocalMetrics {
		logrus.Info(m)
	}
}

//func updateLinks(links []*Link, graphId string) {
//	for _, v := range links {
//		t := &ConfigGraphLink{
//			LinkNode: &LinkNode{
//				Name:            v.NodeFromName,
//				DeviceInterface: v.NodeFrom,
//			},
//			To:      v.NodeToName,
//			GraphId: graphId,
//		}
//		upsertConfigGraph(t)
//	}
//}

//
//func upsertConfigGraph(data *ConfigGraphLink) {
//	data.LinkId = MD5(strings.Join([]string{data.Name, data.DeviceInterface, data.To}, "-"))
//	//data.Action = 3
//	//count, er := data.Count(db.ConfigGraphLinkCol, bson.M{"link_id": data.LinkId, "graph_id": data.GraphId})
//	//if er != nil {
//	//	_ = level.Error(g.Logger).Log("module", "net", "msg", er.Error())
//	//	return
//	//}
//	//if count == 0 {
//	//	_, er = data.Add(db.ConfigGraphLinkCol, data)
//	//	if er != nil {
//	//		_ = level.Error(g.Logger).Log("module", "net", "msg", er.Error())
//	//	}
//	//	return
//	//}
//	return
//}
