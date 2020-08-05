FROM golang:1.14-alpine

WORKDIR /opt/manticore
COPY go.mod .
COPY main.go .

RUN go build . && \
    ls -l 

EXPOSE 8086

CMD ["/opt/manticore/manticore-gosdk-issue"


