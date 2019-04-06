# Dockerfile for making Algernon serve HTTP on port 3000, in development mode
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

# Prepare directories
COPY --from=gobuilder /tmp /tmp
VOLUME /srv/algernon
VOLUME /etc/algernon
WORKDIR /srv/algernon

# Expose port 3000 for HTTP
EXPOSE 3000

# -c assumes no files will be added or removed, for a slight increase in speed
# --domain makes Algernon look for a folder named the same as the domain it serves
# --server turns off interactive and debug mode
# --cachesize sets a file cache size, in bytes
# --prod makes Algernon serve HTTP on port 80 and HTTPS+HTTP/2 on port 443
# --cert and --key is for setting the HTTPS certificate
#
# Other parameters that might be of interest is "--addr", ":3000" together with
# "--server" but without "--prod" for serving only HTTP on port 3000
#
# "--log", "/var/log/algernon.log" can be used for logging errors
#
# "--dev" enables debug mode, uses regular HTTP, enables Bolt and sets the cache mode to "dev".
# "--autorefresh" enables the autorefresh feature where pages are refreshed upon file save.
#
# The final parameter is the directory to serve, for instance /srv/algernon
#
# For "--domain" to work, there should be at least a /srv/algernon/localhost directory.
#
ENTRYPOINT ["/bin/algernon", "--domain", "--server", "--dev", "--autorefresh", "--dir", "/srv/algernon", ":3000"]
CMD ["/bin/algernon"]
