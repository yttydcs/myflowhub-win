// 本文件实现 `transport` 后端服务中与 `codec` 相关的辅助逻辑。

package transport

import sdktransport "github.com/yttydcs/myflowhub-sdk/transport"

type Message struct {
	Action string `json:"action"`
	Data   any    `json:"data,omitempty"`
}

func EncodeMessage(action string, data any) ([]byte, error) {
	return sdktransport.EncodeMessage(action, data)
}
