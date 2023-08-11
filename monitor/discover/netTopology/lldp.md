### SNMP 获取LLDP信息前提
- 目标开启LLDP功能
```
    在系统视图下执行lldp enable命令全局使能LLDP功能，缺省情况下，接口下的LLDP功能与全局下的LLDP功能状态保持一致。
```

- 开启MIB视图包含LLDP-MIB

```
    执行snmp-agent mib-view included iso-view iso命令创建包含所有MIB节点的MIB视图iso-view。
    执行snmp-agent community { read | write } community-name mib-view iso-view命令配置MIB视图iso-view为网管使用的MIB视图。
    执行snmp-agent sys-info version all命令配置交换机启用所有SNMP版本。
```