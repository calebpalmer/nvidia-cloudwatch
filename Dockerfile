FROM golang:1.13 as builder

WORKDIR /build
copy . .
RUN go build

FROM nvidia/cuda:10.2-base
WORKDIR /nvidia-cloudwatch
COPY --from=builder /build/nvidia-cloudwatch .

CMD ["./nvidia-cloudwatch"]