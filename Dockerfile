# Imagen única en la raíz del monorepo para Railway (evita Railpack sin contexto Go).
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
ENV TZ=America/Guatemala
COPY --from=build /server ./server
EXPOSE 8080
CMD ["./server"]
