+++
slug = "docker-root-ownership"
date = 2020-10-03
visibility = "published"
+++

# Avoid Docker root ownership

On Linux, the docker daemon typically runs as root. This is troublesome when using 
Docker to generate files to place on the host system using a bind mount. I ran
into this issue generating `.deb` packages to deploy a server binary. I 
generated the `.deb` packages in a Docker file using FPM and then placed them
onto the host machine. Since Docker runs as root, the generated `.deb` files are
also owned by root, which means I can't delete them as a normal user without 
`sudo`. As a quick example:

```shell
docker run --rm \
    --mount 'type=bind,source=/tmp,destination=/tmp' \
    alpine \
    /bin/sh -c 'echo "docker owned" > /tmp/docker-owned.txt' 
```

The resulting file permissions show the Docker container created file is indeed
owned by root:

```shell
$ ls -alh /tmp/docker-owned.txt
-rw-r--r-- 1 root root 13 Oct  3 00:30 /tmp/docker-owned.txt
```

Luckily, we can control the user ID (UID) and group ID (GID) that docker uses in the Docker 
image by creating the user in the Dockerfile with a user ID that matches the host 
user. Docker shares the same user ID space as the host machine, so if we run
a Docker container with the same user ID as the current user, any files created
by the container will be owned by the current user. To start, we need a 
Dockerfile to create the user with the a UID that matches the current host user's UID.

```Dockerfile
FROM alpine

# Default to root if build args aren't set.
ARG USER_ID=0
ARG GROUP_ID=0
ARG USER_NAME=root

# Create the user matching the build arg USER_ID.
RUN set -eux; \
  if [ $USER_ID != 0 ]; then \
    adduser --disabled-password --gecos '' --uid $USER_ID $USER_NAME; \
  fi;

# Run this image using the user name associated with the same USER_ID passed
# via a build arg variable.
USER $USER_NAME
```

We'll create the image by passing in the current user ID, group ID, and 
user name into the Docker build command:

```shell
docker build \
    --build-arg USER_ID="$(id -u)" \
    --build-arg GROUP_ID="$(id -g)" \
    --build-arg USER_NAME="user" \
    --tag tmp-docker-root-ownership - < /tmp/nonroot.Dockerfile
```

Finally, verify that the files created in the container are owned by the current 
user:

```shell
docker run --rm \                
    --mount 'type=bind,source=/tmp,destination=/tmp' \
    tmp-docker-root-ownership \
    /bin/sh -c 'echo "docker owned" > /tmp/docker-owned-new.txt'

ls -alh /tmp/docker-owned-new.txt 
-rw-r--r-- 1 joe joe 13 Oct  3 00:51 /tmp/docker-owned-new.txt
```

## Debian setup

The Debian setup is similar to Alpine with some added niceties for sudo taken from 
StackOverflow: [How to use sudo in a non-root Docker container](https://stackoverflow.com/a/48329093/30900)

```Dockerfile
FROM debian

# Default to root if build args aren't set.
ARG USER_ID=0
ARG GROUP_ID=0
ARG USER_NAME=root

# Create the user, granting sudo permisions.
RUN set -eux; \
  if [ $USER_ID != 0 ]; then \
    addgroup --gid $GROUP_ID $USER_NAME; \
    adduser --disabled-password --gecos '' --uid $USER_ID \
        --gid $GROUP_ID $USER_NAME; \
    adduser $USER_NAME sudo; \
    echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers; \
  fi;

# Run this image using the user name associated with the same USER_ID passed
# via a build arg variable.
USER $USER_NAME
```


```shell
docker build \
    --build-arg USER_ID="$(id -u)" \
    --build-arg GROUP_ID="$(id -g)" \
    --build-arg USER_NAME="user" \
    --tag tmp-debian-docker-root-ownership - \
    < /tmp/debian-non-root.Dockerfile

docker run --rm \                
    --mount 'type=bind,source=/tmp,destination=/tmp' \
    tmp-debian-docker-root-ownership \
    /bin/sh -c 'echo "docker owned" > /tmp/docker-owned-debian.txt'

ls -alh /tmp/docker-owned-new-debian.txt
-rw-r--r-- 1 joe joe 13 Oct  3 00:59 /tmp/docker-owned-debian.txt
```

::: preview https://stackoverflow.com/a/48329093/30900
How to use sudo in non-root Docker container

Normally, docker containers are run using the user root. I'd like to use a 
different user, which is no problem using docker's USER directive. But this 
user should be able to use sudo inside the container. This command is missing.

The other answers didn't work for me. I kept searching and found a blog post 
that covered how a team was running non-root inside of a docker container.

```Dockerfile
RUN apt-get update \
 && apt-get install -y sudo

RUN adduser --disabled-password --gecos '' docker
RUN adduser docker sudo
RUN echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

USER docker

# this is where I was running into problems with 
# the other approaches
RUN sudo apt-get update 
```
:::
