# Docker

These files can be used for running Algernon as a docker container.

build_prod.sh and run_prod.sh is for building and running the production image for Algernon. This image will serve both HTTP and HTTPS+HTTP/2.
Please adjust the Dockerfile for your needs. In particular, the caching is too agressive if you have a dynamic web application.
You might want to drop the "-c" flag unless you are only serving static files.

build_dev.sh and run_dev.sh is for building and running the development image. It can also take an argument which is either a directory to serve or an .alg file to serve.

Make sure ports are open in your firewall if you are serving anything remotely.

Adjust the configuration and scripts to your needs.
