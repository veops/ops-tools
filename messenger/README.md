# Messenger

messenger是一个简单轻量的消息发送服务，支持邮件、微信、飞书、钉钉，同时支持配置动态修改，使用简单灵活

## 安装

### 源码
```bash
cd messenger
sh build.sh
cp conf/confTemplate.yaml conf/conf.yaml # edit your config
./messenger
```

### docker
```bash
cd messenger
cp conf/confTemplate.yaml conf/conf.yaml # edit your config
docker build --tag messenger .
docker run -d --name messenger -p 8888:8888 -v $(pwd)/conf:/messenger/conf --restart=always messenger 
```

## API

### 发送消息

请求方式：POST

请求地址：http://127.0.0.1:8888/v1/message

参数说明：

| 参数   | 是否必须 | 类型   | 说明                                                   |
| :----- | :------- | :----- | :----------------------------------------------------- |
| sender | 是       | string | sender名称：发送消息具体sender的名称，对应conf中的name |
| msgtype | 是       | string | 消息内容类型：每种消息发送方式支持多种消息内容类型，各发送方式支持的消息内容类型参考如下 <br> email: text/plain text/html <br> wechatBot: [微信机器人](https://developer.work.weixin.qq.com/document/path/99110#%E6%B6%88%E6%81%AF%E7%B1%BB%E5%9E%8B%E5%8F%8A%E6%95%B0%E6%8D%AE%E6%A0%BC%E5%BC%8F)  &emsp;&emsp;&emsp;&nbsp; wechatApp：[微信应用](https://developer.work.weixin.qq.com/document/path/90236#%E6%B6%88%E6%81%AF%E7%B1%BB%E5%9E%8B) <br>  feishuBot：[飞书机器人](https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot#5a997364) &emsp;&emsp;&emsp;&nbsp; feishuApp：[飞书应用](https://open.feishu.cn/document/server-docs/im-v1/message-content-description/create_json#3c92befd) <br> dingdingBot：[钉钉机器人](https://open.dingtalk.com/document/orgapp/custom-robot-access#title-72m-8ag-pqw) &emsp;&emsp; dingdingApp：[钉钉应用](https://open.dingtalk.com/document/orgapp/types-of-messages-sent-by-robots?spm)|
|content|是|string|消息内容：邮件可直接填写内容字符串，其他消息内容本身具有结构，传入其JSON序列化之后的字符串，如微信应用的文本消息填写`{"content":"my content"}`序列化后字符串|
|title|否|string|消息标题：仅用于 email 类型|
|tos|否|[]string|接收人列表：发送邮件、应用消息时需要填写|
|ccs|否|[]string|抄送人列表：仅用于 email 类型|
|extra|否|string|额外参数：通常情况下您只需要关注消息内容类型和其内容发送人，但是当您需要传递一些额外参数时，比如微信应用开启重复检查和检查时间间隔，可以将extra设置为`{"enable_duplicate_check":1, "duplicate_check_interval": 1800}`序列化后字符串|
|sync|否|bool|同步发送：默认情况下，发送请求接受成功即返回200，消息会异步发送，若sync为true则会同步等待消息发送结果并返回|
|simple|否|bool|简单内容：默认情况下，消息内容是json字符串（参考content参数），对于简单的消息类型text和markdown可设置simple=true，此时content仅填写内容字符串本身即可，如`my content`。<br>支持的消息类型（msgtype）<br>text: wechatBot wechatApp feishuBot feishuApp dingdingBot dingdingApp<br>markdown: wechatBot wechatApp dingdingBot dingdingApp
|ats|否|[]string|@列表： 使用@all代表@所有人; 支持wechatBot, feishuBot, dingdingBot |

返回结果：
```json
// 正常 httpStatusCode==200
{
  "msg": "ok"
}

// 异常 httpStatusCode!=200
{
  "msg": "xxxx"
}
```

请求示例：

### curl
```bash
curl  -X POST \
  'http://localhost:8888/v1/message' \
  --header 'Content-Type: application/json' \
  --data-raw '{
  "sender": "yourSenderName",
  "msgtype": "text",
  "content": "{\"content\":\"一行文本内容\"}"
}'
```

### python
```python
import json
import requests

reqUrl = "http://localhost:8888/v1/message"

response = requests.post(reqUrl, json={
    "sender": "yourSenderName",
    "msgtype": "text",
    "content": json.dumps({
        "content": "一行文本内容1"
    })
})

print(response.status_code)
```

### golang
```golang
package main

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

func main() {
	reqUrl := "http://localhost:8888/v1/message"

	content, _ := json.Marshal(map[string]any{
		"content": "一行文本内容2",
	})
	resp, err := resty.New().R().
		SetBody(map[string]any{
			"sender":  "yourSenderName",
			"msgtype": "text",
			"content": string(content),
		}).Post(reqUrl)

	fmt.Println(err, resp.StatusCode())
}
```

### 更新配置

请求方式：POST PUT DELETE

请求地址：http://127.0.0.1:8888/v1/senders

参数说明：

| 参数（请求体）   | 是否必须 | 类型   | 说明                                                   |
| :----- | :------- | :----- | :----------------------------------------------------- |
|body|是|json|请求body为您的sender配置，如`{"wechatBot": [{"name": "yourSenderName", "url": "https://xxx"}]`<br>POST：同类型配置会被全部覆盖<br>PUT：同类型同名称的配置会被更新，新配置将被添加<br>DELETE：同类型同名称配置将被删除|

返回结果：
```json
// 正常 httpStatusCode==200
{
  "msg": "ok"
}

// 异常 httpStatusCode!=200
{
  "msg": "xxxx"
}
```

### 查询用户ID

请求方式：POST

请求地址：http://127.0.0.1:8888/v1/uid/getbyphone

参数说明：

| 参数   | 是否必须 | 类型   | 说明          |
| :----- | :------- | :----- | :------------ |
|sender|是|string|查询用户id时使用的sender名称|
|phone|是|string|手机号|

返回结果：
```json
// 正常 httpStatusCode==200
{
  "uid": "xxxxxxxxxxxxx",
  "msg": "ok"
}

// 异常 httpStatusCode!=200
{
  "msg": "xxxx"
}
```

### 鉴权

当配置文件中开启auths鉴权配置后，请求需要加入鉴权信息，目前支持三种鉴权方式.
#### IP
发送请求的客户端ip需匹配pattern

#### token
发送请求中需要添加请求头 X-Token = token in your yaml config

#### sign
签名鉴权需要添加请求头 X-TS = 当前unix秒数时间戳 X-Nonce = 随机内容 X-Sign = 根据签名算法生成的签名

签名算法步骤为
1. 将ts和nonce信息加入body中 body["ts"] = X-TS, body["nonce"] = X-Nonce
2. 将body中的键值对按键排序后拼接
3. 使用配置文件中的secret计算sha256的值，并将结果进行base64编码，如 secret=666时，步骤二中结果为 
4. 设置请求头中的 X-Sign = 步骤三结果
```golang
// golang 签名示例
secret := "666"
body := map[string]any{
"sender":  "wechatBot",
"msgtype": "text",
"content": "{\"content\":\"一行文本内容\"}",
}
body["ts"] = "1695005697" // cast.ToString(time.Now().Unix())
body["nonce"] = "123"
keys := lo.Keys(body)
sort.Strings(keys)

// content{"content":"一行文本内容"}msgtypetextnonce123senderwechatBotts1695005697
kvStr := strings.Join(lo.Map(keys, func(k string, _ int) string { return fmt.Sprintf("%s%s", k, body[k]) }), "")

mac := hmac.New(sha256.New, []byte(secret))
_, _ = mac.Write([]byte(kvStr))

// nAW4/1vz8EjdJEVXqTevmX7yBOzQtUti1Z2TIgAxogc=
sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
```

## 配置说明

yaml配置文件定义了
1. app 服务配置ip的、端口
2. auths 鉴权方式。多种鉴权方式同时配置时，按配置先后进行检查，满足任意一种方式即通过鉴权。支持的鉴权方式为
   - ip
   - token
   - sign签名
3. senders 具体发送方式。senders支持动态增删，即再服务已经启动的情况下可以直接修改senders列表，服务会持续读取最新的改动。支持的发送方式类型
   - email
   - wechatBot
   - wechatApp
   - feishuBot
   - feishuApp
   - dingdingBot
   - dingdingApp

```yaml
app:
  ip:
  port: 8888

auths:
  # - type: ip
  #   pattern: 192.168.*.*

  # - type: token
  #   token: your token

  # - type: sign
  #   secret: your secret

senders:
  email:
    # - name: yourSenderName1
    #   host: mail.xxx.com
    #   port: 25
    #   account: test@xxx.com
    #   password: #无密码时留空即可
    #   tls: "false"
  wechatBot:
    # - name: yourSenderName2
    #   url: https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxxxx
  wechatApp:
    # - name: yourSenderName3
    #   corpid: xxxx
    #   agentid: xxxx
    #   corpsecret: xxxx
  feishuBot:
    # - name: yourSenderName4
    #   url: https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxx
  feishuApp:
    # - name: yourSenderName5
    #   app_id: cli_xxxx
    #   app_secret: xxxx
  dingdingBot:
    # - name: yourSenderName6
    #   url: https://oapi.dingding.com/robot/send?access_token=xxxx
    #   token: xxxx #仅加密方式为加签时填写
  dingdingApp:
    # - name: yourSenderName7
    #   appKey: xxxx
    #   appSecret: xxxx
    #   robotCode: xxxx
```

## 自定义发送

通常情况下，以上7中方式能满足大部分需求，但是如果你想要定制自己的sender，可以按如下步骤进行开发

1. 在send目录下创建你的sender文件，如mysender.go
2. 定义mysender结构体并实现sender接口
   ```golang
   type sender interface {
	   send(*message) error
	   getConf() map[string]string
   }
   ```
3. 新增init方法将你的sender注册到后台goroutine中，registered的key mysender即为你的sender的类型，可以在配置文件中使用。通常建议将文件名、结构体名、类型名保持一直
   ```golang
   func init() {
	   registered["mysender"] = func(conf config) sender {
	       return &wechatBot{conf: conf}
	   }
   }
   ```