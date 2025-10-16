FROM golang:1.24
WORKDIR /workspace

# Download and extract IBM MQ redistributable client
# Using version 9.4.1.0 which is compatible with the latest mq-golang library
RUN apt-get update && apt-get install -y wget tar && \
    wget -q https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqdev/redist/9.4.1.0-IBM-MQC-Redist-LinuxX64.tar.gz && \
    mkdir -p /opt/mqm && \
    tar -xzf 9.4.1.0-IBM-MQC-Redist-LinuxX64.tar.gz -C /opt/mqm && \
    rm 9.4.1.0-IBM-MQC-Redist-LinuxX64.tar.gz && \
    apt-get remove -y wget && \
    apt-get autoremove -y && \
    rm -rf /var/lib/apt/lists/*

# Set MQ environment
ENV MQ_HOME=/opt/mqm
ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-I${MQ_HOME}/inc"
ENV CGO_LDFLAGS="-L${MQ_HOME}/lib64 -Wl,-rpath=${MQ_HOME}/lib64"
ENV LD_LIBRARY_PATH="${MQ_HOME}/lib64"

# Install Docker CLI for testcontainers
RUN apt-get update && apt-get install -y docker.io && rm -rf /var/lib/apt/lists/*

# Copy only necessary files for dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Run tests
CMD ["go", "test", "-tags", "mqclient", "-v", "./bundles/ibm-mq/..."]