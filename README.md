[English](README_en.md)
# ops-tools

运维过程中一些常用的实践和代码，将一些运维中的小工具不断加入进来，避免重复造轮子

### 独立组件

#### golang
[zabbix2tsdb](zabbix2tsdb/readme.md)  将zabbix的监控数据转换为prometheus格式输出

[netTopology](netTopology) 基于SNMP协议自动发现局域网网络拓扑关系,输出拓扑数据结构

[messenger]([messenger/README.md](https://github.com/veops/messenger)) 简单易用的消息发送服务


#### python
[secret](secret/README.md)  封装好的敏感数据存储工具, 自实现或者对接vault

[auto_discover](auto_discover/README.md)  实例自动发现，主要用于cmdb自发现插件中


### TODO
[netTopology]() 基于现有输出的结构化数据绘制网络拓扑图


---
_**欢迎关注公众号(维易科技OneOps)，关注后可加入微信群，进行产品和技术交流。**_

![公众号: 维易科技OneOps](docs/images/wechat.jpg)





