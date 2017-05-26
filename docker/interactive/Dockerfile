# Dockerfile for making Algernon serve HTTP on port 3000, in development mode

FROM golang
MAINTAINER Alexander F Rødseth <xyproto@archlinux.org>

RUN go get github.com/xyproto/algernon
RUN ln -s "$GOPATH/bin/algernon" /usr/bin/algernon

VOLUME /srv/algernon
VOLUME /etc/algernon

WORKDIR /srv/algernon

# "--domain" makes Algernon look for a folder named the same as the domain it serves
# "--dev" enables debug mode, uses regular HTTP, enables Bolt and sets the cache mode to "dev".
# "--autorefresh" enables the autorefresh feature where pages are refreshed upon file save.
# "--log", "/var/log/algernon.log" can be used for logging errors
#
# The final parameter is the directory or file to serve, for instance /srv/algernon

CMD ["algernon", "--domain", "--dev", "--autorefresh", "--addr", ":4000"]

EXPOSE 4000
