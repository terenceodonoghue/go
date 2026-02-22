FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY pkg/ pkg/
COPY services/solar-svc/ services/solar-svc/
RUN CGO_ENABLED=0 go build -o /solar-svc ./services/solar-svc

FROM alpine:3
RUN apk add --no-cache ca-certificates
COPY --from=build /solar-svc /solar-svc
EXPOSE 8082
ENTRYPOINT ["/solar-svc"]
