FROM golang:1.19.1-bullseye

WORKDIR /src

# download dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# build and install model-hub executable
COPY . .
RUN go build -ldflags "-s -w" -o /model-hub

FROM nvidia/cuda:12.1.1-runtime-ubuntu22.04
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends python3.10 python3-pip sudo curl
RUN pip install --no-cache-dir requests==2.29.0

WORKDIR /bin
COPY --from=0 /model-hub /bin/model-hub
COPY ./worker.py /bin/worker.py

ENV CONFIG_PATH="/etc/config.yaml"
ENV SERVER_PORT="8080"
ENV METRICS_DISPLAY_FREQUENCY="30"
ENV WORKERS_LOADING_STRATEGY="parallel"


ENTRYPOINT ["model-hub"]
