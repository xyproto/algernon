This utility can generate a Dockerfile, given an Algernon application (.alg file)

Example usage:

    alg2docker hello.alg Dockerfile 'John Bob' 'john@thebobs.cx'

The docker image can then be built with:

    docker build -t hello .

And you can run it with:

    docker run -v `pwd`/config:/etc/algernon --rm --publish 80:80 --publish 443:443 hello

Note that the resulting Docker image tries to serve the application as fast as possible and use caching aggressively. Change the options in the Dockerfile if you wish to enable the auto-refresh feature, disable caching or enable the debug mode.

The resulting Docker image will include the application itself, but not the SSL keys used for HTTPS+HTTP/2. They are named `cert.pem` and `key.pem` and needs to be placed in the `config` directory (if using the docker command above).
