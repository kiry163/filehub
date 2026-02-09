FROM node:20-alpine AS web-builder
WORKDIR /app/web-ui
COPY web-ui/package*.json ./
RUN npm ci
COPY web-ui/ ./
RUN npm run build

FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /app/web-ui/dist ./web/dist
ENV CGO_ENABLED=1
RUN go build -o filehub cmd/filehub/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite-libs
WORKDIR /app
COPY --from=builder /app/filehub ./filehub
COPY config.yaml ./config.yaml
EXPOSE 8080
CMD ["./filehub"]
