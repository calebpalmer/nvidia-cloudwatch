package cloudwatch

import (
	"errors"
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

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func find(slice []*float64, val *float64) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

// mergeMetrics, merges two sets of metrics.
func mergeMetrics(m1 *cloudwatch.MetricDatum, m2 *cloudwatch.MetricDatum) *cloudwatch.MetricDatum {

	m1.Timestamp = m2.Timestamp
	m1.Dimensions = m2.Dimensions
	m1.MetricName = m2.MetricName
	m1.StorageResolution = m2.StorageResolution
	m1.Unit = m2.Unit

	if m1.Values == nil {
		m1.Values = make([]*float64, 0, int(*m2.StorageResolution))
		m1.Values = nil
	}
	if m1.Counts == nil {
		m1.Counts = make([]*float64, 0, int(*m2.StorageResolution))
	}

	k, found := find(m1.Values, m2.Value)
	if found == false {
		m1.Values = append(m1.Values, m2.Value)
		count := 1.0
		m1.Counts = append(m1.Counts, &count)
	} else {
		newCount := *m1.Counts[k] + 1.0
		m1.Counts[k] = &newCount
	}

	return m1
}

// getMetrics gets metrics to be pushed to cloudwatch
func getMetrics(instance *string, resolution int64) []*cloudwatch.MetricDatum {
	time := time.Now()

	devices, err := nvidia.GetDevices()
	if err != nil {
		panic(err)
	}

	instanceDimension := "Instance"
	gpuDimension := "GPU"
	gpuUtilizationMetricName := "GPUUtilization"
	memoryUsedMetricName := "MemoryUsed"
	memoryFreeMetricName := "MemoryFree"
	percentUnit := cloudwatch.StandardUnitPercent
	memoryUnit := cloudwatch.StandardUnitMegabytes

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
			Unit:              &memoryUnit,
			Value:             &memoryUsed,
		}
		metrics = append(metrics, &memoryUsedMetric)

		memoryFree := float64(*device.FreeMemory)
		memoryFreeMetric := cloudwatch.MetricDatum{
			Timestamp:         &time,
			Dimensions:        dimensions,
			MetricName:        &memoryFreeMetricName,
			StorageResolution: &resolution,
			Unit:              &memoryUnit,
			Value:             &memoryFree,
		}
		metrics = append(metrics, &memoryFreeMetric)

	}
	return metrics
}

// logMetrics pushes metrics into Cloudwatch.
func logMetrics(svc *cloudwatch.CloudWatch,
	metrics []*cloudwatch.MetricDatum) {

	for _, m := range metrics {
		err := m.Validate()
		if err != nil {
			panic(err)
		}
	}

	namespace := "nvidia-cloudwatch"
	_, err := svc.PutMetricData(&cloudwatch.PutMetricDataInput{
		MetricData: metrics,
		Namespace:  &namespace,
	})

	if err != nil {
		log.Panic(err)
	}
}

// logAllMetrics logs a list of list of metrics.
func logAllMetrics(svc *cloudwatch.CloudWatch, metricsList [][]*cloudwatch.MetricDatum) {
	for _, metrics := range metricsList {
		logMetrics(svc, metrics)
	}
}

// StartExporter starts the process of pushing metrics
func StartExporter() {
	region := os.Getenv("AWS_REGION")
	sess, err := createSession(&region)
	if err != nil {
		log.Panic(err)
	}

	// the period at which we collect and send metrics.
	period := 60
	p := os.Getenv("PERIOD")
	if p != "" {
		period, err = strconv.Atoi(p)
		if err != nil {
			log.Panic(err)
		}
	}

	resolution := 60
	r := os.Getenv("RESOLUTION")
	if r != "" {
		resolution, err = strconv.Atoi(r)
		if err != nil {
			log.Panic(err)
		}
	}

	if !(resolution == 60 || resolution == 1) {
		panic(errors.New("Resolution must be 1 or 60"))
	}

	if period < resolution {
		panic(errors.New("Period must be greater than or equal to resolution."))
	}

	instanceName := getInstance()

	svc := cloudwatch.New(sess)

	nvidia.Init()
	defer nvidia.Shutdown()

	if resolution == 60 {

		lastLogTime := time.Now().Truncate(time.Second)
		metrics := make([]*cloudwatch.MetricDatum, 3, 3)
		for i, _ := range metrics {
			metrics[i] = &cloudwatch.MetricDatum{}
		}

		for {
			nextTime := time.Now().Truncate(time.Second)
			nextTime = nextTime.Add(time.Duration(resolution) * time.Second)

			newMetrics := getMetrics(&instanceName, int64(resolution))

			// merge them.  We assume that they are in the same order.
			for i, _ := range metrics {
				metrics[i] = mergeMetrics(metrics[i], newMetrics[i])
			}

			if int(time.Now().Truncate(time.Second).Sub(lastLogTime)/time.Second) >= period {
				go logMetrics(svc, metrics)
				lastLogTime = time.Now().Truncate(time.Second)
				metrics = make([]*cloudwatch.MetricDatum, 3, 3)
				for i, _ := range metrics {
					metrics[i] = &cloudwatch.MetricDatum{}
				}
			}

			time.Sleep(time.Until(nextTime))
		}
	} else {
		// 1 second resolution
		lastLogTime := time.Now().Truncate(time.Second)
		metrics := make([][]*cloudwatch.MetricDatum, 0, 60)

		for {

			nextTime := time.Now().Truncate(time.Second)
			nextTime = nextTime.Add(time.Duration(resolution) * time.Second)

			metrics = append(metrics, getMetrics(&instanceName, int64(resolution)))
			if int(time.Now().Truncate(time.Second).Sub(lastLogTime)/time.Second) >= period {
				go logAllMetrics(svc, metrics)
				lastLogTime = time.Now().Truncate(time.Second)
				metrics = make([][]*cloudwatch.MetricDatum, 0, 60)
			}

			time.Sleep(time.Until(nextTime))
		}
	}
}
