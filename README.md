# Proxy over gRPC

pog-server is a HTTP proxy which uses gRPC for sending bytes:
    User <== (HTTP proxy) ==> pog client <=== (gRPC) ===> pog server <=== (HTTP proxy) ===> destination server

# Simple example of use

The pog server:
```bash
$ go run ./grpcproxy/server
2024/06/15 13:28:02 proxy-over-grpc server, version: dev
2024/06/15 13:28:02 starting on port 8080
2024/06/15 13:28:02 PID: 43396
2024/06/15 13:28:02 waiting for termination signal...
pog: ifconfig.me:443 anonymous HTTPS [::1]:51225 [2024-06-15T13:28:12+03:00] OK
```

The pog client:
```bash
$ SERVER_ADDR=localhost:8080 INSECURE=1 go run ./grpcproxy/client
2024/06/15 13:28:07 proxy-over-grpc client listening address :18080
2024/06/15 13:28:07 PID: 43580
2024/06/15 13:28:07 waiting for termination signal...
pog: ifconfig.me:443 anonymous HTTPS 127.0.0.1:51226 [2024-06-15T13:28:12+03:00] 200
```

A user of the HTTP proxy:
```bash
$ curl -i --proxy http://localhost:18080 https://ifconfig.me
HTTP/1.1 200 OK
Date: Sat, 15 Jun 2024 10:28:12 GMT
Transfer-Encoding: chunked

HTTP/2 200
date: Sat, 15 Jun 2024 10:28:12 GMT
content-type: text/plain
content-length: 14
access-control-allow-origin: *
via: 1.1 google
alt-svc: h3=":443"; ma=2592000,h3-29=":443"; ma=2592000

1.136.246.102
```

Here the user reaches the destination URL ifconfig.me via the HTTP proxy at https://localhost:18080 .

# How to build

[Go](https://go.dev/) programming language version > 1.21 is required.

```bash
go build -o server ./grpcproxy/server
go build -o client ./grpcproxy/client
go build -o client ./grpcproxy/genauthitem
```

# How to build Docker image

```bash
$ docker build --platform linux/amd64 --build-arg GRPCPROXY_COMMIT=$(git rev-parse --short HEAD) \
  -f grpcproxy/server/Dockerfile -t pog-server:dev .
```