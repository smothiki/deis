FROM mesosphere/marathon

RUN apt-get update && apt-get install -yq golang git mercurial

ENV GOPATH /go
WORKDIR /go/src/github.com/deis/deis/mesos/marathon
ENTRYPOINT ["/bin/sh", "-c"]
CMD /go/bin/marathon

ADD . /go/src/github.com/deis/deis/mesos/marathon
RUN CGO_ENABLED=0 go get -a -ldflags '-s' github.com/deis/deis/mesos/marathon
