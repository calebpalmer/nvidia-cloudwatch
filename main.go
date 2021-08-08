package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/calebpalmer/nvidia-cloudwatch/pkg/cloudwatch"
	"github.com/calebpalmer/nvidia-cloudwatch/pkg/nvidia"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	//"github.com/calebpalmer/nvidia-cloudwatch/pkg/cloudwatch"
)

var gpuMemoryGaugeVec = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "nvidia_gpu_mem",
	},
	[]string{
		"uuid",
		"model",
	})

var gpuTotalMemoryGaugeVec = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "nvidia_gpu_mem_total",
	},
	[]string{
		"uuid",
		"model",
	})

var gpuUsageGaugeVec = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "nvidia_gpu_usage",
	},
	[]string{
		"uuid",
		"model",
	})

func recordMetrics() {

	go func() {
		for {
			devices, err := nvidia.GetDevices()
			if err != nil {
				panic(err)
			}

			for _, device := range devices {
				gpuMemoryGaugeVec.With(prometheus.Labels{"uuid": device.UUID, "model": *device.Model}).Set(float64(*device.UsedMemory))
				gpuTotalMemoryGaugeVec.With(prometheus.Labels{"uuid": device.UUID, "model": *device.Model}).Set(float64(*device.TotalMemory))
				gpuUsageGaugeVec.With(prometheus.Labels{"uuid": device.UUID, "model": *device.Model}).Set(float64(*device.GPUUtilization))
			}

			time.Sleep(2 * time.Second)
		}
	}()
}

func main() {
	nvidia.Init()
	defer nvidia.Shutdown()

	startCloudwatchExporter := flag.Bool("--cloudwatch-exporter", false, "Enable cloudwatch exporter")
	if *startCloudwatchExporter {
		fmt.Println("Starting cloudwatch exporter.")
		go cloudwatch.StartExporter()
	}

	recordMetrics()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
