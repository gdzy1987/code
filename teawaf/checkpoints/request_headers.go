package checkpoints

import (
	"net/http"
	"strings"
)

type RequestHeadersCheckpoint struct {
	Checkpoint
}

func (this *RequestHeadersCheckpoint) RequestValue(req *http.Request, param string) (value interface{}, err error) {
	var headers = []string{}
	for k, v := range req.Header {
		for _, subV := range v {
			headers = append(headers, k+": "+subV)
		}
	}
	value = strings.Join(headers, "\n")
	return
}
