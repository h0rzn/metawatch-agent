package stats

import "github.com/docker/docker/api/types"

type Disk struct {
	Read  float64
	Write float64
}

func NewDisk(disk types.BlkioStats) {

}
