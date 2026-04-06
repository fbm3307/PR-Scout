# Stage 1: Build dashboard
FROM node:22-alpine AS dashboard-builder
WORKDIR /app/web/dashboard
COPY web/dashboard/package.json web/dashboard/package-lock.json ./
RUN npm ci
COPY web/dashboard/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS go-builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o bin/pr-scout ./cmd/pr-scout

# Stage 3: Runtime
FROM alpine:3.22
RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=go-builder /app/bin/pr-scout .
COPY --from=dashboard-builder /app/web/dashboard/dist ./web/dashboard/dist
RUN mkdir -p deploy/config

EXPOSE 8080
ENTRYPOINT ["./pr-scout", "--dashboard-dir", "./web/dashboard/dist"]
