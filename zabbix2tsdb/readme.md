tools for converting zabbix data to tsdb format, the new output contain metric,labels,value and timestamp
# output
## influxdb
```shell
sys_cpu_idle,core=cpu1,env=prod {V}=12.34 1690892492000000000
```

## prometheus
> output as follows
```shell
system_cpu_idle{core="cpu1",env="prod"} 12.34 1690892492000000000
```

# how to use
```shell
  -k, --acceptKeys strings   需要同步的Key,通配符匹配(*). 例如:如需要同步'system.'开头的key_,则配置'system.*' (default [system.*])
  -a, --address string       zabbix api请求地址。为防止带宽占用以及网络延迟问题，我们强烈推荐将节点选择在与zabbix api
                             服务在同一台机器上。如 http://127.0.0.1:8080/api_jsonrpc.php,可以写完整地址,也可以直接省去后缀(api_jsonrpc.php),
                             如写http://127.0.0.1:8080
  -v, --apiVersion string    api版本 (default "2.0")
  -c, --cluster string       zabbix集群名称, 当采集多个zabbix集群,且不同集群存在相同的主机名(ip),可以避免数据混乱 (default "0")
  -f, --dataFormat string    data format that you want to convert to, you can choose 'prometheus' or 'influxdb', (default "influxdb")
  -g, --groups strings       需要同步的group分组 (default [Linux servers,Zabbix servers,Virtual machines])
  -i, --interval int         同步时间间隔,单位秒. 防止对zabbix服务器造成太大压力,系统允许的最小时间间隔为30秒 (default 60)
  -p, --password string      允许通过api访问数据的用户对应的密码,推荐使用环境变量 (default "zabbix")
  -u, --user string          允许通过api访问数据的用户名, 推荐使用环境变量 (default "Admin")
```