package metrics

import "github.com/docker/docker/api/types"

type Disk struct {
	Read  float64 `json:"read" bson:"disk_read"`
	Write float64 `json:"write" bson:"disk_write"`
}

func NewDisk(disk types.BlkioStats) *Disk {
	var read, write uint64
	for _, io := range disk.IoServiceBytesRecursive {
		if len(io.Op) > 0 {
			switch io.Op[0] {
			case 'r', 'R':
				read = read + io.Value
			case 'w', 'W':
				write = write + io.Value
			}
		} else {
			continue
		}
	}

	return &Disk{
		Read:  float64(read),
		Write: float64(write),
	}
}
