FROM golang:1.14

WORKDIR /go/src/hashthing/

COPY *.go ./
RUN go get -d -v
RUN go install -v

# Copy over to empty image
FROM alpine
COPY --from=0 /go/bin/hashthing /hashthing
# VOLUME /tmp
ENTRYPOINT ["/hashthing"]
