package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/pflag"
)

const (
	MaxIdleConnections int = 20
	RequestTimeout     int = 30
	ApiSuffix              = "api_jsonrpc.php"
	defaultKey             = "{V}"
)

func Init() {
	KeyPattern = regexp.MustCompile(`^([0-9a-zA-Z_.]*)\\[(.*)\\]$`)
	ReplMetricPattern = regexp.MustCompile(`([^a-zA-Z0-9:_]+?)`)
	ReplParamsPattern = regexp.MustCompile(`([," =]+?)`)
	//for _, v := range zabbixConfig.AccetpKeys{
	//	if p, er := regexp.Compile(v); er == nil{
	//		AcceptKeysPattern = append(AcceptKeysPattern, p)
	//	}
	//}
	zabbixConfig.Cluster = ReplParamsPattern.ReplaceAllString(zabbixConfig.Cluster, "_")
	address := zabbixConfig.Address
	if !strings.HasSuffix(address, ApiSuffix) {
		zabbixConfig.Address = fmt.Sprintf("%s/%s", strings.TrimSuffix(address, "/"), ApiSuffix)
	}
	zApi = NewZabbixClient()
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
}

type Host struct {
	Locker sync.RWMutex
	Ids    []string
}

type Config struct {
	Groups     map[string]struct{}
	Cluster    string
	Address    string
	User       string
	Password   string
	ApiVersion string
	AccetpKeys []string
	Interval   int64
	DataFormat string
}

var (
	zabbixConfig = Config{Groups: map[string]struct{}{}}
	zApi         *ZabbixApi
	signals      = make(chan os.Signal, 1)
	// hostIdHost 通过item.get获取到hostId之后，需要查找对应的host
	hostIdHost sync.Map
	// globalHostIds 之所以使用map，是因为多个groupId可能包含相同的hostId
	//globalHostIds sync.Map
	hosts       = Host{Locker: sync.RWMutex{}}
	groupNameId sync.Map

	// KeyPattern key_中提取metric,params
	KeyPattern *regexp.Regexp
	// ReplMetricPattern metric名称转换
	ReplMetricPattern *regexp.Regexp
	// ReplParamsPattern 用来转换params,hostName
	ReplParamsPattern *regexp.Regexp
	AcceptKeysPattern []*regexp.Regexp

	ConnectError = fmt.Errorf("connectError")

	// hostQueryBatch 批量查询host采集项时，控制一次性查询量,防止数据过多
	hostQueryBatch int = 150
)

func (h *Host) Set(hostIds map[string]struct{}) {
	tmp := make(map[string]struct{})
	h.Locker.Lock()
	defer h.Locker.Unlock()
	// delete expired hostId
	for i := 0; i < len(h.Ids); i++ {
		if _, ok := hostIds[h.Ids[i]]; !ok {
			h.Ids = append(h.Ids[:i], h.Ids[i+1:]...)
			i--
		} else {
			tmp[h.Ids[i]] = struct{}{}
		}
	}
	// add host
	for hostId := range hostIds {
		if _, ok := tmp[hostId]; !ok {
			h.Ids = append(h.Ids, hostId)
		}
	}
}

// createHTTPClient for connection re-use
func createHTTPClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost: MaxIdleConnections,
		},
		Timeout: time.Duration(RequestTimeout) * time.Second,
	}
	return client
}

type ZabbixApi struct {
	User       string
	Password   string
	Address    string
	ApiVersion string
	Auth       string
	Client     *http.Client
}

func NewZabbixClient() *ZabbixApi {
	c := createHTTPClient()
	return &ZabbixApi{
		User:       zabbixConfig.User,
		Password:   zabbixConfig.Password,
		Address:    zabbixConfig.Address,
		ApiVersion: zabbixConfig.ApiVersion,
		Client:     c,
	}
}

func (z *ZabbixApi) Login() error {
	r, er := json.Marshal(map[string]interface{}{
		"jsonrpc": z.ApiVersion,
		"method":  "user.login",
		"params": map[string]string{
			"user":     z.User,
			"password": z.Password,
		},
		"id": 1,
	})
	if er != nil {
		return er
	}
	body := bytes.NewBuffer(r)
	req, er := http.NewRequest("GET", z.Address, body)
	if er != nil {
		return er
	}
	req.Header.Set("Content-Type", "application/json-rpc")
	resp, er := z.Client.Do(req)
	if er != nil {
		return ConnectError
	}
	defer resp.Body.Close()
	result, er := io.ReadAll(resp.Body)
	if er != nil {
		return er
	}
	respData := make(map[string]interface{})
	er = json.Unmarshal(result, &respData)
	if er != nil {
		return er
	}
	if auth, ok := respData["result"]; ok {
		z.Auth = auth.(string)
	}
	if z.Auth == "" {
		return fmt.Errorf("获取认证秘钥为空")
	}
	return nil
}

func (z *ZabbixApi) GroupIds() error {
	r, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": z.ApiVersion,
		"method":  "hostgroup.get",
		"params": map[string]interface{}{
			"output": []string{"name", "groupid"},
		},
		"auth": z.Auth,
		"id":   1,
	})
	body := bytes.NewBuffer(r)
	req, er := http.NewRequest("GET", z.Address, body)
	if er != nil {
		return er
	}
	req.Header.Set("Content-Type", "application/json-rpc")
	resp, er := z.Client.Do(req)
	if er != nil {
		return ConnectError
	}
	defer resp.Body.Close()
	result, er := io.ReadAll(resp.Body)
	if er != nil {
		return er
	}
	respData := make(map[string]interface{})
	er = json.Unmarshal(result, &respData)
	if er != nil {
		return er
	}
	if result, ok := respData["result"]; ok {
		for _, v := range result.([]interface{}) {
			groupNameId.Store(v.(map[string]interface{})["name"].(string),
				v.(map[string]interface{})["groupid"].(string))
		}
	}
	return nil
}

func (z *ZabbixApi) HostIds(groupIds []string) error {
	r, er := json.Marshal(map[string]interface{}{
		"jsonrpc": z.ApiVersion,
		"method":  "host.get",
		"params": map[string]interface{}{
			"output": []string{"hostid", "host"},
			//"output": "extend",
			"groupids": groupIds,
		},
		"auth": z.Auth,
		"id":   1,
	})
	if er != nil {
		return er
	}
	body := bytes.NewBuffer(r)
	req, er := http.NewRequest("GET", z.Address, body)
	if er != nil {
		return er
	}
	req.Header.Set("Content-Type", "application/json-rpc")
	resp, er := z.Client.Do(req)
	if er != nil {
		return er
	}
	defer resp.Body.Close()
	result, er := ioutil.ReadAll(resp.Body)
	if er != nil {
		return er
	}
	respData := make(map[string]interface{})
	er = json.Unmarshal(result, &respData)
	if er != nil {
		return er
	}
	tmp := map[string]struct{}{}
	if result, ok := respData["result"]; ok {
		for _, v := range result.([]interface{}) {
			if hostId, ok := v.(map[string]interface{})["hostid"]; ok {
				tmp[hostId.(string)] = struct{}{}
				if host, ok := v.(map[string]interface{})["host"]; ok {
					if _, ok := hostIdHost.Load(host); !ok {
						hostIdHost.Store(hostId, host)
					}
				}
			}
		}
	}
	hosts.Set(tmp)
	return nil
}

func (z *ZabbixApi) SmartItems(hostIds []string) {
	var hostIdsGroups [][]string
	groupNumber := int(math.Ceil(float64(len(hostIds)) / float64(hostQueryBatch)))
	for i := 0; i < groupNumber; i++ {
		startIndex := i * hostQueryBatch
		endIndex := (i + 1) * hostQueryBatch
		if endIndex > len(hostIds) {
			endIndex = len(hostIds)
		}
		hostIdsGroups = append(hostIdsGroups, hostIds[startIndex:endIndex])
	}
	for _, subHostIds := range hostIdsGroups {
		er := z.Items(subHostIds)
		if er != nil {
			if er == ConnectError {
				_ = z.Login()
			}
		}
	}

}

func (z *ZabbixApi) Items(hostIds []string) error {
	r, er := json.Marshal(map[string]interface{}{
		"jsonrpc": z.ApiVersion,
		"method":  "item.get",
		"params": map[string]interface{}{
			//"output": []string{"hostid"},
			"output":                 []string{"key_", "hostid", "lastvalue", "lastclock", "value_type"},
			"hostids":                hostIds,
			"search":                 map[string]interface{}{"key_": zabbixConfig.AccetpKeys},
			"searchWildcardsEnabled": true,
			"searchByAny":            true,
		},
		"auth": z.Auth,
		"id":   1,
	})
	if er != nil {
		return er
	}
	body := bytes.NewBuffer(r)
	req, er := http.NewRequest("GET", z.Address, body)
	if er != nil {
		return er
	}
	req.Header.Set("Content-Type", "application/json-rpc")
	resp, er := z.Client.Do(req)
	if er != nil {
		return ConnectError
	}
	defer resp.Body.Close()
	result, er := io.ReadAll(resp.Body)
	if er != nil {
		return er
	}
	respData := make(map[string]interface{})
	er = json.Unmarshal(result, &respData)
	if er != nil {
		return er
	}
	if result, ok := respData["result"]; ok {
		for _, v := range result.([]interface{}) {
			er := processMetric(v.(map[string]interface{}))
			if er != nil {
				_, _ = fmt.Fprintln(os.Stderr, er.Error())
			}
		}
	}
	return nil
}

//func isAcceptedKey(key string) bool {
//	for _, p := range AcceptKeysPattern {
//		if p.MatchString(key) {
//			return true
//		}
//	}
//	return false
//}

func processMetric(item map[string]interface{}) error {
	dataStr := ""
	// only numeric/float values can convert to prometheus
	//dataStr := "%s,c=%s,__endpoint__=%s,params=%s {val}=%s %s000000000"
	if item["value_type"] == "0" || item["value_type"].(string) == "3" {
		if _, ok := item["key_"]; !ok {
			return fmt.Errorf("no key_")
		}
		// step 1: metric
		//if !isAcceptedKey(item["key_"].(string)){
		//	return nil
		//}
		res := KeyPattern.FindStringSubmatch(item["key_"].(string))
		if len(res) > 1 {
			dataStr = ReplMetricPattern.ReplaceAllString(res[1], "_")
		} else {
			dataStr = ReplMetricPattern.ReplaceAllString(item["key_"].(string), "_")
		}
		// step 2 tags: zabbix cluster
		dataStr = fmt.Sprintf("%s,c=%s", dataStr, zabbixConfig.Cluster)
		// step3 tags: hostName
		if hostName, ok := hostIdHost.Load(item["hostid"].(string)); ok {
			dataStr = fmt.Sprintf("%s,__endpoint__=%s",
				dataStr, ReplParamsPattern.ReplaceAllString(hostName.(string), "_"))
		}
		// step 4 tags: params
		if len(res) >= 2 && res[0] != "" {
			dataStr = fmt.Sprintf("%s,p=%s", dataStr, ReplParamsPattern.ReplaceAllString(res[2], "\\$1"))
		}
		// step5 处理值
		if _, ok := item["lastvalue"]; !ok {
			return nil
		}
		dataStr = fmt.Sprintf("%s %s=%v", dataStr, defaultKey, item["lastvalue"])
		// step5 处理时间
		if _, ok := item["lastclock"]; ok && len(item["lastclock"].(string)) == 10 {
			dataStr = fmt.Sprintf(" %s %v000000000", dataStr, item["lastclock"])
		} else {
			return nil
		}
	}
	if dataStr != "" {
		if zabbixConfig.DataFormat == "prometheus" {
			s := convertInfluxToPrometheus(dataStr)
			for _, v := range s {
				fmt.Println(v)
			}
		} else {
			fmt.Println(dataStr)
		}
	}
	return nil
}

func convertInfluxToPrometheus(influxData string) []string {
	var data []string
	splitData := strings.Split(influxData, " ")
	measurementTagSet := strings.Split(splitData[0], ",")
	measurement := measurementTagSet[0]
	tagSet := measurementTagSet[1:]

	fieldSet := strings.Split(splitData[1], ",")
	var timestamp string
	if len(splitData) == 3 {
		timestamp = splitData[2]
	}

	labelSet := make([]string, len(tagSet))
	var t []string
	for i, tag := range tagSet {
		t = strings.SplitN(tag, "=", 2)
		if len(t) == 2 {
			labelSet[i] = fmt.Sprintf("%s=\"%s\"", t[0], t[1])
		}
	}
	for _, val := range fieldSet {
		t = strings.SplitN(val, "=", 2)
		if len(t) == 2 {
			if t[0] != defaultKey {
				measurement = fmt.Sprintf(`%s_%s`, measurement, t[0])
			}
			data = append(data, fmt.Sprintf(`%s{%s} %s %s`, measurement, strings.Join(labelSet, ","), t[1], timestamp))
		}
	}
	return data
}

func configParse() {
	//var groups []string
	pflag.StringVarP(&zabbixConfig.Cluster, "cluster", "c", "0",
		"zabbix集群名称, 当采集多个zabbix集群,且不同集群存在相同的主机名(ip),可以避免数据混乱")
	pflag.StringVarP(&zabbixConfig.Address, "address", "a", "",
		fmt.Sprintf(`zabbix api请求地址。为防止带宽占用以及网络延迟问题，我们强烈推荐将节点选择在与zabbix api
服务在同一台机器上。如 http://127.0.0.1:8080/%[1]s,可以写完整地址，也可以直接省去后缀(%[1]s)，
如写http://127.0.0.1:8080`, ApiSuffix))
	pflag.StringVarP(&zabbixConfig.User, "user", "u", "Admin", "允许通过api访问数据的用户名, 推荐使用环境变量")
	pflag.StringVarP(&zabbixConfig.Password, "password", "p", "zabbix", "允许通过api访问数据的用户对应的密码,推荐使用环境变量")
	groups := pflag.StringSliceP("groups", "g",
		[]string{"Linux servers", "Zabbix servers", "Virtual machines"}, "需要同步的group分组")
	pflag.StringVarP(&zabbixConfig.ApiVersion, "apiVersion", "v", "2.0", "api版本")
	//pflag.StringSliceVarP(&zabbixConfig.AccetpKeys, "acceptKeys", "k",
	//	[]string{"system\\..*"}, "需要同步的Key,正则匹配")
	pflag.StringSliceVarP(&zabbixConfig.AccetpKeys, "acceptKeys", "k",
		[]string{"system.*"}, "需要同步的Key,通配符匹配(*). 例如:如需要同步'system.'开头的key_,则配置'system.*'")
	pflag.Int64VarP(&zabbixConfig.Interval, "interval", "i", int64(60),
		"同步时间间隔,单位秒. 防止对zabbix服务器造成太大压力,系统允许的最小时间间隔为30秒")
	pflag.StringVarP(&zabbixConfig.DataFormat, "dataFormat", "f", "influxdb",
		"data format that you want to convert to, you can choose 'prometheus' or 'influxdb', default is influxdb")
	pflag.Parse()
	if zabbixConfig.Address == "" {
		_, _ = fmt.Fprintln(os.Stderr, "address 不能为空,我们推荐防止带宽占用，将节点选择在与zabbix api 服务在同一台机器上")
		os.Exit(0)
	}
	if len(*groups) > 0 {
		for _, group := range *groups {
			zabbixConfig.Groups[group] = struct{}{}
		}
	}
	if zabbixConfig.Interval < 30 {
		zabbixConfig.Interval = 30
	}
	if s := os.Getenv("password"); s != "" {
		zabbixConfig.Password = s
	}
	if s := os.Getenv("user"); s != "" {
		zabbixConfig.User = s
	}
}

func updateGroupAndHost() {
	er := zApi.GroupIds()
	if er != nil {
		//_, _ = fmt.Fprintln(os.Stderr, fmt.Sprintf("update groups failed:%s", er))
		_, _ = fmt.Fprintf(os.Stderr, "update groups failed:%s\n", er)
		// login again
		er := zApi.Login()
		if er != nil {
			_, _ = fmt.Fprintf(os.Stderr, "login failed:%s\n", er)
		}
	}
	groupIds := []string{}
	for groupName := range zabbixConfig.Groups {
		if groupId, ok := groupNameId.Load(groupName); ok {
			groupIds = append(groupIds, groupId.(string))
		}
	}
	if len(groupIds) == 0 {
		return
	}
	er = zApi.HostIds(groupIds)
	if er != nil {
		_, _ = fmt.Fprintf(os.Stderr, "update hostid failed:%s\n", er)
		// login again
		er := zApi.Login()
		if er != nil {
			_, _ = fmt.Fprintf(os.Stderr, "login failed:%s\n", er)
		}
	}
}

func GetItems() {
	if len(hosts.Ids) == 0 {
		return
	}
	zApi.SmartItems(hosts.Ids)
}

func Loop(ctx context.Context) {
	updateGroupAndHost()
	GetItems()
	ticker := time.NewTicker(time.Minute * 5)
	itemTicker := time.NewTicker(time.Second * time.Duration(zabbixConfig.Interval))
	for {
		select {
		case <-ticker.C:
			updateGroupAndHost()
		case <-itemTicker.C:
			GetItems()
		case <-signals:
			return
		case <-ctx.Done():
			return
		}
	}
}

func main() {
	configParse()
	Init()
	er := zApi.Login()
	if er != nil {
		_, _ = fmt.Fprintf(os.Stderr, "login failed:%s\n", er)
		os.Exit(1)
	}
	Loop(context.Background())
}
