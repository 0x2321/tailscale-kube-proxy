FROM golang:1.24 AS build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN go vet -v
RUN go test -v

RUN CGO_ENABLED=0 go build -o /go/bin/tskp

FROM gcr.io/distroless/static-debian12
LABEL org.opencontainers.image.source="https://github.com/0x2321/tailscale-kube-proxy"

COPY --from=build /go/bin/tskp /
ENTRYPOINT ["/tskp"]
CMD ["serve"]