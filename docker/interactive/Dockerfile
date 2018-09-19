# Dockerfile for making Algernon serve HTTP on port 4000, in development mode
FROM golang:alpine as gobuilder
MAINTAINER Alexander F Rødseth <xyproto@archlinux.org>

# Prepare the needed files
COPY . "$GOPATH/src/github.com/xyproto/algernon"
WORKDIR "$GOPATH/src/github.com/xyproto/algernon"

# Build Algernon
RUN GO111MODULE=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /go/bin/algernon

# Start from scratch, only copy in the Algernon executable
FROM scratch
COPY --from=gobuilder /go/bin/algernon /go/bin/algernon

# Prepare directories
COPY --from=gobuilder /tmp /tmp
VOLUME /srv/algernon
VOLUME /etc/algernon
WORKDIR /srv/algernon

# Expose port 4000 for HTTP
EXPOSE 4000

# "--domain" makes Algernon look for a folder named the same as the domain it serves
# "--dev" enables debug mode, uses regular HTTP, enables Bolt and sets the cache mode to "dev".
# "--autorefresh" enables the autorefresh feature where pages are refreshed upon file save.
# "--log", "/var/log/algernon.log" can be used for logging errors
#
# The final parameter is the directory or file to serve, for instance /srv/algernon
ENTRYPOINT ["/go/bin/algernon", "--domain", "--dev", "--autorefresh", "--addr", "--dir", "/srv/algernon", ":4000"]
CMD ["/go/bin/algernon"]
