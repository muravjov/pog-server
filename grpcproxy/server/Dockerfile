FROM golang:1.21-bookworm as builder

WORKDIR /app

# Retrieve application dependencies.
# This allows the container build to reuse cached dependencies.
# Expecting to copy go.mod and if present go.sum.
#COPY go.mod go.sum ./
#RUN --mount=target=. go mod download
RUN --mount=target=. go install .

ARG GRPCPROXY_COMMIT

COPY . .
RUN CGO_ENABLED=0 go build -mod=readonly -v -o /bin/server -ldflags "-X main.Version=1.0.0_${GRPCPROXY_COMMIT:-dockerfile_empty_value}" ./grpcproxy/server
RUN CGO_ENABLED=0 go build -mod=readonly -v -o /bin/client -ldflags "-X main.Version=1.0.0_${GRPCPROXY_COMMIT:-dockerfile_empty_value}" ./grpcproxy/client
RUN CGO_ENABLED=0 go build -mod=readonly -v -o /bin/genauthitem -ldflags "-X main.Version=1.0.0_${GRPCPROXY_COMMIT:-dockerfile_empty_value}" ./grpcproxy/genauthitem

FROM alpine:latest as certimage
RUN apk --no-cache add ca-certificates tzdata

FROM scratch

# x509: certificate signed by unknown authority, see
# https://stackoverflow.com/a/52979541/4825871
COPY --from=certimage /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# https://medium.com/@mhcbinder/using-local-time-in-a-golang-docker-container-built-from-scratch-2900af02fbaf
COPY --from=certimage /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=builder /bin/server /server
COPY --from=builder /bin/client /client
COPY --from=builder /bin/genauthitem /genauthitem

ENTRYPOINT ["/server"]
CMD []

