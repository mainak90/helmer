FROM golang:alpine

MAINTAINER mainak90
# Run it with these params : docker run -e ELEPHANTSQL_URL="<Your-PSQL-URL>" \
# --mount type=bind,source=/tmp/charts,target=/tmp/charts \
# -v /Users/mdhar/.kube/config:/root/.kube/config -u root -d f31975325b52
# Set necessary environmet variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Move to working directory /build
WORKDIR /build

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY . .

# Build the application
RUN go build -o main .

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

COPY static/ /dist/static

# Copy binary from build to main folder
RUN cp /build/main .

# Export necessary port
EXPOSE 8900

# Command to run when starting the container
CMD ["/dist/main"]