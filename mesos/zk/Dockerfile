FROM jplock/zookeeper:3.4.6

RUN apt-get update && apt-get install -yq golang git mercurial

ENV GOPATH /go
WORKDIR /go/src/github.com/deis/deis/mesos/zk
ENTRYPOINT ["/bin/sh", "-c"]
CMD /go/bin/zk

ADD . /go/src/github.com/deis/deis/mesos/zk
RUN CGO_ENABLED=0 go get -a -ldflags '-s' github.com/deis/deis/mesos/zk
