FROM golang:1.9

# glide tool install
RUN curl https://glide.sh/get | sh

# There's a go-wrapper tool installed in the golang container that works for
# simple projects. But we want to use glide to install our deps, so create the
# root structure we need to have go think our armory package is valid. Use
# glide to install the deps and the go build to make an executable.
WORKDIR /go/src/github.com/spinnaker/roer
COPY . .
RUN glide i
RUN go build cmd/roer/main.go

# Copy the executable to something named more sane
RUN cp /go/src/github.com/spinnaker/roer/main /roer

ENTRYPOINT ["/roer"]
