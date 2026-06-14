# Start by building the application.
FROM golang:1.26 AS build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN make build

# Now copy it into our base image.
FROM gcr.io/distroless/static-debian13
COPY --from=build /go/src/app/tailscale-kube-proxy /app
CMD ["/app"]