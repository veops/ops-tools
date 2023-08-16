package discover

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	. "netTopology/internal"
)

const (
	RedisKeyDiscoverLinks = "Net::Discover::Links"
	RedisKeyCompleteLinks = "Net::Complete::Links"
)

type CollectorConfig struct {
	Id      string `json:"id"`
	Name    string
	Gateway string `json:"gateway"`
	Nodes   []*NodeSnmp
}

type Discover interface {
	Collectors(gateway string) ([]*CollectorConfig, error)
	Topology(config *CollectorConfig) (*TopologyGraph, error)
	UploadRecord(data *TopologyGraph) error
	UploadTsRecord(target string, data map[string]map[string]int64)
	Nodes() []string
}

func tidyGraph(graph *TopologyGraph, portMapIfIndex map[string]map[string]int64,
	portMapDes map[string]map[string]string) {
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
	var ips []string
	for index, v := range graph.Nodes {
		if v1, ok := NodeInfo.Load(v.Name); ok {
			v2 := v1.(map[int64]*IfIndexInfo)
			d := map[string][]string{}
			for _, v3 := range v2 {
				ips = []string{}
				for _, v4 := range v3.ToIps {
					v4 = ToHardwareAddr(v4)
					if ip, ok := MacIps.Load(v4); ok {
						ips = append(ips, ip.([]string)...)
					} else {
						ips = append(ips, v4)
					}
				}
				d[v3.Des] = ips
			}
			graph.Nodes[index].DesIps = d
		}
	}
}

//
//func upsertConfigGraph(data *models.ConfigGraphLink) {
//	data.LinkId = internal.MD5(strings.Join([]string{data.Name, data.DeviceInterface, data.To}, "-"))
//	data.Action = 3
//	count, er := data.Count(db.ConfigGraphLinkCol, bson.M{"link_id": data.LinkId, "graph_id": data.GraphId})
//	if er != nil {
//		_ = level.Error(g.Logger).Log("module", "net", "msg", er.Error())
//		return
//	}
//	if count == 0 {
//		_, er = data.Add(db.ConfigGraphLinkCol, data)
//		if er != nil {
//			_ = level.Error(g.Logger).Log("module", "net", "msg", er.Error())
//		}
//		return
//	}
//	return
//}
//
//func updateLinks(links []*Link, graphId string) {
//	for _, v := range links {
//		t := &models.ConfigGraphLink{
//			LinkNode: &models.LinkNode{
//				Name:            v.NodeFromName,
//				DeviceInterface: v.NodeFrom,
//			},
//			To:      v.NodeToName,
//			GraphId: graphId,
//		}
//		upsertConfigGraph(t)
//	}
//}

func start(md Discover, ident string) error {
	logrus.Info("start discover network topology")
	collectors, er := md.Collectors(ident)
	if er != nil {
		return er
	}

	for _, v := range collectors {
		logrus.Info("start collect from ", v.Id)
		graph, er1 := md.Topology(v)
		if er1 != nil {
			er = errors.Wrap(er, er1.Error())
			continue
		}
		er1 = md.UploadRecord(graph)
		if er != nil {
			er = errors.Wrap(er, er1.Error())
		}
	}
	logrus.Info("end discover network topology")
	logrus.Info("the last auto discover topology graph is as follows")
	logrus.Info("---------------------------------------------------")
	TG.Range(func(key, value any) bool {
		v := value.(*TopologyGraph)
		s, _ := json.MarshalIndent(v, "", " ")
		logrus.Info(string(s))
		return true
	})
	logrus.Info("---------------------------------------------------")
	return nil
}

func NetworkMonitorInit(cancel <-chan os.Signal, md Discover) {
	go func() {
		//time.Sleep(time.Duration(rand.New(rand.NewSource(time.Now().Unix())).Intn(24)) * time.Hour)
		er := start(md, "")
		if er != nil {
			logrus.Error(er)
		}
		ticker := time.NewTicker(time.Hour * 24)
		for {
			select {
			case <-cancel:
				return
			case <-ticker.C:
				er := start(md, "")
				if er != nil {
					logrus.Error(er)
				}
			}
		}
	}()
	// 网络流量每分钟更新一次
	go func() {
		nodes := md.Nodes()
		for _, node := range nodes {
			go func(node string) {
				data, err := ifStatistics(node)
				if err != nil {
					logrus.Error(err)
				}
				md.UploadTsRecord(node, data)
			}(node)
		}
		//ticker := time.NewTicker(time.Second * 10)
		ticker := time.NewTicker(time.Minute)
		for {
			select {
			case <-cancel:
				return
			case <-ticker.C:
				nodes := md.Nodes()
				for _, node := range nodes {
					go func(node string) {
						data, err := ifStatistics(node)
						if err != nil {
							logrus.Error(err)
						}
						md.UploadTsRecord(node, data)
					}(node)
				}
			}
		}
	}()
}

//
//func UniversalAction(action string, req string) (resp, nextAction string, err error) {
//	switch action {
//	case "Net.UploadRecord":
//		data := TopologyGraph{}
//		er := json.Unmarshal([]byte(req), &data)
//		if er != nil {
//			logrus.Error(er)
//			return
//		}
//		md := MasterDiscover("master")
//		er = md.UploadRecord(&data)
//		return "", "", md.UploadRecord(&data)
//	case "Net.Collectors":
//		data := map[string]string{}
//		md := MasterDiscover("master")
//		if ident, ok := data["ident"]; ok {
//			d, er := md.Collectors(ident)
//			if er == nil {
//				s, er := json.Marshal(d)
//				if er == nil {
//					return string(s), "", nil
//				}
//			}
//			if er != nil {
//				logrus.Error(er)
//				return
//			}
//		}
//	}
//	return "", "", nil
//}
