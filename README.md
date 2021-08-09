# nvidia-cloudwatch
Exports nvidia device information via a prometheus metrics endpoint and optionally to AWS cloudwatch.

## Installation

### Binary

	go get https://github.com/calebpalmer/nvidia-cloudwatch.git
	$GOPATH/bin/nvidia-cloudwatch

#### With Cloudwatch exporting
The typical AWS environment variables are used to configure the environment for pushing the cloudwatch metrics.  eg:

	export AWS_ACCESS_KEY_ID=keyid
	export AWS_SECRET_ACCESS_KEY=secretkey
	export AWS_REGION=ca-central-1
	$GOPATH/bin/nvidia-cloudwatch -cloudwatch

The default resolution and period for sending metrics are 60 seconds but can be configured with the PERIOD and RESOLUTION environment variables.

### Helm Chart
Clone this repository and install the chart:

	git clone git@github.com:calebpalmer/nvidia-cloudwatch.git
	helm upgrade --install nvidia-cloudwatch

#### With Cloudwatch exporting

	microk8s.helm3 upgrade --install nvidia-cloudwatch helm/nvidia-cloudwatch -f - <<EOF
	cloudwatch:
	  enabled: true
	  region: us-east-1
	  accessKeyId: xxxx
	  secretAccessKey: xxxx
	  period: 60
	  resolution: 60
	EOF
