# Dockerfile for only using the Lua interpreter in Algernon
FROM golang:alpine as gobuilder
MAINTAINER Alexander F. Rødseth <xyproto@archlinux.org>

# Prepare the needed files
COPY . "$GOPATH/src/github.com/xyproto/algernon"
WORKDIR "$GOPATH/src/github.com/xyproto/algernon"

# Build Algernon
RUN GO111MODULE=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /bin/algernon

# Start from scratch, only copy in the Algernon executable
FROM scratch
COPY --from=gobuilder /bin/algernon /bin/algernon
COPY --from=gobuilder /tmp /tmp
WORKDIR /tmp

# Only start the Lua interpreter
CMD ["/bin/algernon", "--conf", "/dev/null", "--lua"]
