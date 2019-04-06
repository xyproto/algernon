# Dockerfile for only using the Lua interpreter in Algernon
FROM golang:alpine as gobuilder
MAINTAINER Alexander F. Rødseth <xyproto@archlinux.org>

# Prepare the needed files
COPY . /algernon
WORKDIR /algernon

# Build Algernon
RUN GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=0 \
    go build \
      -mod=vendor \
      -a \
      -installsuffix cgo \
      -ldflags="-w -s" \
      -o /bin/algernon

RUN apk add upx && upx /bin/algernon

# Start from scratch, only copy in the Algernon executable
FROM scratch
COPY --from=gobuilder /bin/algernon /bin/algernon
COPY --from=gobuilder /tmp /tmp

# Only start the Lua interpreter
ENTRYPOINT ["/bin/algernon", "--lua"]
CMD ["/bin/algernon", "--lua"]
