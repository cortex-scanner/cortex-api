FROM golang:1.25-alpine AS build

WORKDIR /build

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags "-s -w" -o /build/bin/cortex-api ./cmd/


FROM scratch
COPY --from=build /build/bin/cortex-api /cortex-api
ENTRYPOINT ["/cortex-api"]