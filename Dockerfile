FROM golang:1.9

WORKDIR /go/src/github.com/spinnaker/roer
COPY . .
RUN go build cmd/roer/main.go

# Copy the executable to something named more sane
RUN cp /go/src/github.com/spinnaker/roer/main /roer

ENTRYPOINT ["/roer"]
