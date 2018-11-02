These are just example keys, for serving HTTPS, HTTP/2 or QUIC with a self-signed certificate.

Here is one of many possible ways to create a self-signed certificate (`key.pem` + `cert.pem`):

    openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 3000 -nodes

Most browsers will alert the user about the cerificate being self-signed.
