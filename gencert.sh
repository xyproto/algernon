#!/bin/sh
# For generating SSL certs, for testing.
# Just press return at all the prompt, but enter "localhost" at Common Name.
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 9999 -nodes
