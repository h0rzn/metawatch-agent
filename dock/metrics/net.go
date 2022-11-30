package metrics

import "github.com/docker/docker/api/types"

type Net struct {
	In  float64 `json:"in" bson:"net_in"`
	Out float64 `json:"out" bson:"net_out"`
}

func NewNet(net map[string]types.NetworkStats) *Net {
	var in, out float64 // rx, tx
	for _, v := range net {
		in += float64(v.RxBytes)
		out += float64(v.TxBytes)
	}
	return &Net{
		In:  in,
		Out: out,
	}
}
