This utility can generate a Dockerfile, given an Algernon application (.alg file)

Example usage:

    alg2docker hello.alg Dockerfile 'John Bob' 'john@thebobs.cx'

The docker image can then be built with:

    docker build -t hello .

And you can run it with:

    docker run -v `pwd`/config:/etc/algernon --rm --publish 80:80 --publish 443:443 hello
