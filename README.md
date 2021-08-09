# nvidia-cloudwatch
Exports nvidia device information via a prometheus metrics endpoint and optionally to AWS cloudwatch.

## Installation

### Container

	docker run calebpalmer/nvidia-cloudwatch

### Helm Chart
Clone this repository and install the chart:

	git clone git@github.com:calebpalmer/nvidia-cloudwatch.git
	helm upgrade --install nvidia-cloudwatch
