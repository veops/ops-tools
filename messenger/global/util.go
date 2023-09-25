package global

import "encoding/json"

func RenderPretty(a any) string {
	bs, _ := json.MarshalIndent(a, "", " ")
	return string(bs)
}
