# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
COPY templates ./templates
COPY static ./static
COPY content ./content
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /eagle-vet .

FROM scratch
COPY --from=build /eagle-vet /eagle-vet
USER 65532:65532
ENV HOST=0.0.0.0 PORT=8080
EXPOSE 8080
ENTRYPOINT ["/eagle-vet"]
