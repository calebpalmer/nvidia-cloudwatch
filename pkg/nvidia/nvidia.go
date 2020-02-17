package nvidia

import (
	"encoding/json"
	"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"
	"log"
)

// Device holds nvidia gpu inforation and metrics
type Device struct {
	UUID           string
	Model          *string
	TotalMemory    *uint64
	UsedMemory     *uint64
	FreeMemory     *uint64
	GPUUtilization *uint
}

// String returns the String represenation for a Device
func (d Device) String() string {
	dJson, _ := json.Marshal(&d)
	return string(dJson)
}

// Init initializes the nvidia package.
func Init() {
	nvml.Init()
	log.Println("nvml initialized")
}

// Shutdown shuts down the nvidia package.
func Shutdown() {
	nvml.Shutdown()
	log.Println("nvml shutdown")
}

// GetDevices gets the nvidia devices
func GetDevices() ([]*Device, error) {
	count, err := nvml.GetDeviceCount()
	if err != nil {
		log.Panicln("Error getting device count:", err)
	}

	devices := make([]*Device, 0, 1)

	for i := uint(0); i < count; i++ {
		nvmlDevice, err := nvml.NewDevice(i)
		if err != nil {
			log.Panicf("Error getting device %d: %v\n", i, err)
		}

		status, err := nvmlDevice.Status()
		if err != nil {
			return nil, err
		}

		devices = append(devices, &Device{
			UUID:           nvmlDevice.UUID,
			Model:          nvmlDevice.Model,
			TotalMemory:    nvmlDevice.Memory,
			UsedMemory:     status.Memory.Global.Used,
			FreeMemory:     status.Memory.Global.Free,
			GPUUtilization: status.Utilization.GPU,
		})
	}

	return devices, nil

}
