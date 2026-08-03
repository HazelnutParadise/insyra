package nn

import "github.com/HazelnutParadise/insyra/accel"

func init() {
	RegisterDeviceMatMul(accel.DeviceMatMul)
}
