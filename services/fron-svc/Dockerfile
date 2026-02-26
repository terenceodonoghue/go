FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY pkg/ pkg/
COPY services/fron-svc/ services/fron-svc/
RUN CGO_ENABLED=0 go build -o /fron-svc ./services/fron-svc

FROM alpine:3
RUN apk add --no-cache ca-certificates
COPY --from=build /fron-svc /fron-svc
EXPOSE 8082
ENTRYPOINT ["/fron-svc"]
