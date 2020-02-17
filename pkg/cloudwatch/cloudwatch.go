package cloudwatch

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/calebpalmer/nvidia-cloudwatch/pkg/nvidia"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// createSession creates an AWS Session
func createSession(region *string) (*session.Session, error) {
	if region == nil || *region == "" {
		*region = "us-east-1"
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: aws.String("us-east-1"),
		},
	})

	if err != nil {
		return nil, err
	}

	return sess, nil
}

// getInstance gets the name of the ec2 instance or NA if not running on an ec2 instance
func getInstance() string {
	resp, err := http.Get("http://169.254.169.254/latest/meta-data/instance-id")
	if err != nil {
		return "NA"
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return string(body)
}

// getMetrics gets metrics to be pushed to cloudwatch
func getMetrics(instance *string, resolution int64) []*cloudwatch.MetricDatum {
	time := time.Now()

	devices, err := nvidia.GetDevices()
	if err != nil {
		panic(err)
	}

	instanceDimension := "Intance"
	gpuDimension := "GPU"
	gpuUtilizationMetricName := "GPUUtilization"
	memoryUsedMetricName := "MemoryUsed"
	memoryFreeMetricName := "MemoryFree"
	percentUnit := cloudwatch.StandardUnitPercent
	bytesUnit := cloudwatch.StandardUnitBytes

	metrics := make([]*cloudwatch.MetricDatum, 0, 3)

	for _, device := range devices {
		dimensions := []*cloudwatch.Dimension{
			&cloudwatch.Dimension{Name: &instanceDimension, Value: instance},
			&cloudwatch.Dimension{Name: &gpuDimension, Value: &device.UUID},
		}

		gpuUtilization := float64(*device.GPUUtilization)
		gpuUtilizationMetric := cloudwatch.MetricDatum{
			Timestamp:         &time,
			Dimensions:        dimensions,
			MetricName:        &gpuUtilizationMetricName,
			StorageResolution: &resolution,
			Unit:              &percentUnit,
			Value:             &gpuUtilization,
		}
		metrics = append(metrics, &gpuUtilizationMetric)

		memoryUsed := float64(*device.UsedMemory)
		memoryUsedMetric := cloudwatch.MetricDatum{
			Timestamp:         &time,
			Dimensions:        dimensions,
			MetricName:        &memoryUsedMetricName,
			StorageResolution: &resolution,
			Unit:              &bytesUnit,
			Value:             &memoryUsed,
		}
		metrics = append(metrics, &memoryUsedMetric)

		memoryFree := float64(*device.UsedMemory)
		memoryFreeMetric := cloudwatch.MetricDatum{
			Timestamp:         &time,
			Dimensions:        dimensions,
			MetricName:        &memoryFreeMetricName,
			StorageResolution: &resolution,
			Unit:              &bytesUnit,
			Value:             &memoryFree,
		}
		metrics = append(metrics, &memoryFreeMetric)

	}
	return metrics
}

// logMetrics pushes metrics into Cloudwatch.
func logMetrics(svc *cloudwatch.CloudWatch,
	metrics []*cloudwatch.MetricDatum) {

	namespace := "nvidia-cloudwatch"
	_, err := svc.PutMetricData(&cloudwatch.PutMetricDataInput{
		MetricData: metrics,
		Namespace:  &namespace,
	})

	if err != nil {
		log.Panic(err)
	}
}

// StartExporter starts the process of pushing metrics
func StartExporter() {
	region := os.Getenv("AWS_REGION")
	sess, err := createSession(&region)
	if err != nil {
		log.Panic(err)
	}

	period := 60

	resolution := 60
	r := os.Getenv("RESOLUTION")
	if r != "" {
		resolution, err = strconv.Atoi(r)
		if err != nil {
			log.Panic(err)
		}
	}

	instanceName := getInstance()

	cloudwatch := cloudwatch.New(sess)

	nvidia.Init()
	defer nvidia.Shutdown()
	for {
		nextTime := time.Now().Truncate(time.Second)
		nextTime = nextTime.Add(time.Duration(period) * time.Second)

		metrics := getMetrics(&instanceName, int64(resolution))
		logMetrics(cloudwatch, metrics)

		time.Sleep(time.Until(nextTime))
	}
}
