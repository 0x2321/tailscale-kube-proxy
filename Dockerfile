# Start by building the application.
FROM golang:1.26 AS build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/bin/app

# Now copy it into our base image.
FROM gcr.io/distroless/static-debian13
COPY --from=build /go/bin/app /
CMD ["/app"]