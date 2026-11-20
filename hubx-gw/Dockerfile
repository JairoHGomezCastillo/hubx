# hubx-gw image. Build context is hubx-gw/ only: services never share code.
FROM golang:1.26.5-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOWORK=off
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/gw ./cmd/gw

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/gw ./
COPY application.properties ./
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/gw"]
