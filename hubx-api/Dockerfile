# hubx-api image: the API server (/app/api) and the migration runner (/app/migrate).
# Build context is hubx-api/ only: services never share code.
FROM golang:1.26.5-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOWORK=off
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/api /out/migrate ./
COPY application.properties ./
USER nonroot:nonroot
EXPOSE 8081
ENTRYPOINT ["/app/api"]
