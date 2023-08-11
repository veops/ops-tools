// Package models /*
package discover

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type NodeSnmp struct {
	Ip        string `json:"ip" bson:"ip"`
	Community string `json:"community" bson:"community"`
	Version   string `json:"version" bson:"version"`
}

type GraphSnmp struct {
	BaseModel `bson:",inline"`
	Id        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name      string             `json:"name" bson:"name"`
	Gateway   string             `json:"gateway" bson:"gateway"`
	Nodes     []*NodeSnmp        `json:"nodes" bson:"nodes"`
	Enable    bool               `json:"enable" bson:"enable"`
	OneAgent  string             `json:"one_agent" bson:"one_agent"`
}

type LinkNode struct {
	Name            string `json:"name" bson:"name"`
	DeviceInterface string `json:"device_interface" bson:"device_interface"`
	// HighSpeed 单位 Mbps
	HighSpeed int `json:"high_speed" bson:"high_speed"`
}

type BaseModel struct {
	CreateTime time.Time `json:"create_time" bson:"create_time"`
	UpdateTime time.Time `json:"update_time" bson:"update_time"`
	Creator    string    `json:"creator" bson:"creator"`
}

type ConfigGraphLink struct {
	*BaseModel `bson:",inline"`
	*LinkNode  `bson:",inline"`
	Id         primitive.ObjectID `json:"-" bson:"_id,omitempty"`
	GraphId    string             `json:"-" bson:"graph_id"`
	// Name 自动生成，根据from,to的名称和type构成
	LinkId  string `json:"link_id" bson:"link_id"`
	To      string `json:"to" bson:"to"`
	Comment string `json:"comment" bson:"comment"`
	// Action 表示改链接的状态
	// 0: 人工删除， 1:人工确认或者添加 3:自动发现
	Action uint8 `json:"action" bson:"action"`
}

//type ConfigGraphNode struct {
//	BaseModel
//	Id               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
//	Name             string             `json:"name" bson:"name"`
//	DeviceInterfaces []*LinkNode        `json:"device_interfaces" bson:"device_interfaces"`
//}

type Metric struct {
	Metric    string            `protobuf:"bytes,1,opt,name=metric,proto3" json:"metric,omitempty"`
	Value     float64           `protobuf:"fixed64,2,opt,name=value,proto3" json:"value,omitempty"`
	Timestamp int64             `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Tags      map[string]string `protobuf:"bytes,4,rep,name=tags,proto3" json:"tags,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Ns        string            `protobuf:"bytes,5,opt,name=ns,proto3" json:"ns,omitempty"`
	Type      string            `protobuf:"bytes,6,opt,name=type,proto3" json:"type,omitempty"`
}
