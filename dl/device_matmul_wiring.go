//go:build !race

package dl

import "github.com/HazelnutParadise/insyra/accel"

func init() {
	RegisterDeviceMatMul(accel.DeviceMatMul)
}
