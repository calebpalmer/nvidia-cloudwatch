package main

import (
	"github.com/calebpalmer/nvidia-cloudwatch/pkg/cloudwatch"
)

func main() {
	cloudwatch.StartExporter()
}
