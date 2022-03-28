# alg2docker

This utility can generate a Dockerfile, given an Algernon application (`.alg` file).

### Step by step usage

1. Have an `.alg` file (for instance `hello.alg`), and also a directory with a `cert.pem` and `key.pem` file that can be used when serving HTTPS.

2. Generate the Dockerfile:

    `./alg2docker -f hello.alg Dockerfile 'John Bob' 'john@thebobs.cx'`

3. Build the Docker image:

    `docker build -t hello .`

The resulting Docker image will include the application itself, but not the SSL keys used for HTTPS.

4. Serve the application using docker and the `cert.pem` and `key.pem` files in `$PWD/config`:

    `docker run -v "$PWD/config":/etc/algernon --publish 80:80 --publish 443:443 --rm hello`

### Tweaks

* The cache settings can be modified after creating the `Dockerfile` by changing the command line arguments that are given to Algernon, at the bottom of the file. One might want to disable caching, enable the auto-refresh feature or enable debug mode.
* When `--domain` is used and a directory `/srv/algernon` corresponds to a valid domain name for the server, like `example.com`, then `/srv/algernon/example.com/` will be served when users vists `example.com`, if Algernon is running on that server.
* Using `--letsencrypt` together with the `--domain` option, on a server that responds to requests for that domain, will use Let's Encrypt for fetching keys and certificates for the HTTPS port, and use CertMagic to serve both HTTP and HTTPS.
