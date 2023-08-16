// Package internal /*
package internal

//import "serinus/pkg/server/models"

type LinkValue struct {
	From   string      `json:"from"`
	To     string      `json:"to"`
	Key    string      `json:"key"`
	Value  interface{} `json:"value"`
	Unit   string      `json:"unit"`
	Alias  string      `json:"alias"`
	Speed  int         `json:"speed"`
	Status interface{} `json:"status"`
}

//type LinkV1 struct {
//	// Name 自动生成，根据from,to的名称和type构成
//	LinkId  string           `json:"link_id"`
//	Node1   *models.LinkNode `json:"node1"`
//	Node2   *models.LinkNode `json:"node2"`
//	Values  []*LinkValue     `json:"values"`
//	Options interface{}      `json:"options"`
//	Status  string           `json:"status"`
//}

type NodeStatus struct {
	Status string `json:"status"`
	Value  string `json:"value"`
	Metric string `json:"metric"`
	Alias  string `json:"alias"`
}

type Node struct {
	// Name must be unique
	Name string `json:"name"`
	// Alias
	Alias string `json:"alias"`
	// Type switch/router/server/internet/firewall/unknown
	Type string `json:"type"`
	// X location x
	X int `json:"x"`
	// Y location y
	Y int `json:"y"`
	// Ips 本机IP
	Ips []string `json:"ips"`
	// Locals 直连ip
	Locals []string `json:"locals"`
	// discoverIps 间接连接ip
	Options    interface{}         `json:"options"`
	Status     string              `json:"status"`
	IfIndexIps map[int64][]string  `json:"if_index_ips"`
	DesIps     map[string][]string `json:"des_ips"`
}

//type TopologyGraphV1 struct {
//	Name          string                                       `json:"name"`
//	Nodes         []*Node                                      `json:"nodes"`
//	Links         []*LinkV1                                    `json:"links"`
//	Location      bool                                         `json:"location"`
//	Illustrate    interface{}                                  `json:"illustrate"`
//	StatusDetails map[string]map[string]map[string]*NodeStatus `json:"status_details"`
//}
