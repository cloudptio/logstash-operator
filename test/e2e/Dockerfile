# Run e2e tests
FROM golang:1.13

ARG E2E_JSON
ENV E2E_JSON $E2E_JSON

WORKDIR /go/src/github.com/cloudptio/logstash-operator

# Create the go test cache directory
RUN mkdir -p /.cache && chmod 777 /.cache
# If a restricted PSP is applied we can't run as root
USER 101

# Download dependencies
COPY ["go.mod", "go.sum","./"]
RUN go mod download

# Copy the source
COPY pkg/ pkg/
COPY config/ config/
COPY test/ test/

ENTRYPOINT ["test/e2e/run.sh"]
