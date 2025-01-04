+++
slug = "docker-patterns"
date = 2020-05-03
visibility = "draft"
+++

# Implementation patterns in Docker

After spending a considerable amount of time using Docker for CI pipelines, I've
noticed a few patterns and developed some guidelines to make my life easier.

## Shell hygiene

We'll start with shell hygiene. Firstly, prefix complex chains of `RUN` commands
with `set -eux` to stop script execution on errors (`errexit`), stop on unset
variables (`nounset`), and trace commands (`x`).

```dockerfile
RUN set -eux \
  && apt-get update \
  && apt-get -y install awscli jq curl ruby ruby-dev \
  && gem install --no-document fpm \
  && curl https://getcaddy.com | bash -s personal
```

NOTE: The efficacy of errexit is a nuanced topic with sharp edges. The
counter-argument against errexit revolves around its [inability][errexit] to
correctly detect all conditions with non-zero exit codes, including relatively
common conditions. I've found errexit more than valuable enough to adorn the top
of my scripts.

[errexit]: http://mywiki.wooledge.org/BashFAQ/105

## Move logic out of a Dockerfile into a script

Dockerfile syntax is horrendous for building scripts past about 5 lines. Instead
of wrestling with `&&` and backslashes, move logic into a script. Then, `COPY`
the script into the Docker image. Docker invalidates based on the hash of the
copied script, so any changes in the copied script force Docker to rebuild that
layer plus all following layers. The benefit of using external scripts are
that you can use Bash instead of the container shell and all the programming
tools with external files, especially shellcheck.

```dockerfile
FROM debian:latest
COPY download_source.sh /usr/local/bin
RUN /usr/local/bin/download_vector.sh
```

## Always update apt cache

Minor versions in Debian package repositories advance quickly into oblivion.
Always prefix `apt-get install` with `apt-get update`. If the base image has a
previously issued `apt-get update` command, its apt cache contains with
available package versions that existed at the creation time of the image.
Debian removes old versions from the package repositories, so `apt-get install`
can fail if the cached version is too old.

```shell
apt-get update
```

## Cache expensive operations in the Docker layer cache

Docker caches each command in a Dockerfile (`RUN`, `COPY`, etc.) as a layer. If
a layer is unchanged, and all the preceding layers are also unchanged, Docker
checks its cache and avoids rerunning the commands if a cache entry exists. This
pattern appears for:

- Downloading dependencies like node_modules, Maven packages, or Go modules.
- Downloading source code from a versioned URL. You cannot use the latest URL
  because the object might change, but the URL does not.

## Robust curl commands

Fetching data with curl has a few pitfalls. The two most common are not
supporting redirects and not failing on 400 or 500 status codes.

To verify, run `curl -v httpbin.org/status/404; echo status: $?`. The exit code
variable `$?` will be set to 0, indicating no error. From curl's perspective, a
0 exit code makes sense. curl successfully received a response from a server
that happened to have a status code other than 200. Use the `--fail` flag to
direct curl to use the exit code 22 when the status code is not 200.

A robust curl command looks like:

```shell
# --fail returns a non-zero exit code on a non-200 status
#   code.
# --location follows redirects instead of returning the
#   redirect HTML
curl --fail --location --output "${out_file}" "${url}"
```

## Extracting single items out of a `.tar.gz` file

Building from source code requires downloading the source code first. A TAR
bundle provides an efficient, versioned mechanism to download source code.

Instead of extracting all files in a compressed tar archive, extract only the
files you need.

```shell
url="https://packages.timber.io/vector/nightly/2020-05-04/vector-amd64.deb"
out="/tmp/vector_amd64.deb"
curl --fail --location --output "${out}" "${url}"
# Extract usr/bin/vector from vector.tar.xz into /tmp/vector by
# 1. chdir into /tmp/
# 2. dropping the first two parts of the path, usr/ and bin/
# 3. only extracting the file usr/bin/vector
tar xvf vector.tar.xz \
    -C /tmp/ \
    --strip-components=2 \
    usr/bin/vector
```

## Alias an image used in multiple build targets

It's convenient to reference a single image by using an alias. This is useful
when pinning to a specific version. Alternately, build arg variables provide a
similar, but slightly longer method.

```dockerfile
ARG DEBIAN_VERSION=sid-20190812-slim

# Alias the base Debian image so we can upgrade it in one spot.
FROM debian:${DEBIAN_VERSION} as debian_pinned

FROM debian_pinned as builder-debian
RUN set -eux \
  && apt-get update \
  && apt-get install ca-certificates git openssh-client
```

## Build a small production image

One way to build a small Docker image is to use Docker build layers (TODO:
name). The idea is to use a wasteful image to build and compile a small binary
that can be isolated on a small image.

```dockerfile
ARG DEBIAN_VERSION=sid-20190812-slim
ARG NODE_VERSION=10.15.3

# Alias the base Debian image so we can upgrade it in one spot.
FROM debian:${DEBIAN_VERSION} as debian

# Builder image with dev tools and useful apt-get defaults.
# This image is used for building binaries that are copied into the
# final image.
FROM debian as builder-debian
ARG DEBIAN_FRONTEND=noninteractive
WORKDIR /builder
RUN set -eux \
  && printf 'APT::Get::Install-Recommends "0";\nAPT::Get::Install-Suggests "0"\n;' > /etc/apt/apt.conf.d/90-no-recommends \
  && printf 'APT::Get::Assume-Yes "true";\n;' > /etc/apt/apt.conf.d/91-assume-yes \
  && mkdir -p /var/log/apt \
  && chown _apt /builder
RUN set -eux \
  && apt-get update \
  && apt-get install ca-certificates git libexpat1 libcurl3-gnutls \
         openssh-client patch postgresql-client \
  # Remove large git binaries that we don't need. Most git executables
  # are symlinks to git or simple shell scripts.
  && find /usr/lib/git-core/ -type f -size +1M -not -path '*/git-remote-http' -delete \
  && rm -rf /usr/share/perl

# Builder image for NPM and node.
FROM node:${NODE_VERSION}-stretch as builder-node
COPY --from=builder-node-prune /go/bin/node-prune /usr/bin/
ARG NODE_MODULES=/usr/local/lib/node_modules
RUN set -eux \
  # Remove unnecessary cruft from NPM's node_modules.
  && rm -rf ${NODE_MODULES}/npm/{man,html,doc,changelogs} \
  && find ${NODE_MODULES} -type f -name '*.min.js' -delete \
  && node-prune ${NODE_MODULES}
COPY remove_unneeded_files.sh /builder/
RUN /builder/remove_unneeded_files.sh

FROM builder-debian as debian-ci-node
COPY --from=builder-node /usr/local/lib/node_modules /usr/local/lib/node_modules
COPY --from=builder-node /usr/local/bin/node /usr/local/bin/
RUN set -eux \
  # Link NPM so it's available on the path.
  && ln -s /usr/local/lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm
COPY remove_unneeded_files.sh /builder/
RUN /builder/remove_unneeded_files.sh
COPY docker_container_cache_bust.txt /
USER root
```

## Docker anti-patterns

Patterns that aren't worth pursuing. The big one is Docker image sizes. Usually,
it's enough to use builder images to produce good-enough sized images. I don't
recommend trying to trim existing images beyond basic maneuvers. Here's a list
of size optimizations I've pursued:

- Removing large git binaries, and perl-based binaries. CI builds necessitate
  git. Git takes about 65MB. A majority of the size is rarely used binaries and
  the perl dependency. Perl is used for interactive commands, which is not need
  in CI. I removed most of the fat with:

  ```shell
  RUN set -eux \
    && apt-get install git \
    # Remove large git binaries that we don't need. Most git executables
    # are symlinks to git or simple shell scripts.
    && find /usr/lib/git-core/ -type f -size +1M -not -path '*/git-remote-http' -delete \
      && rm -rf /usr/share/perl
  ```

- I attempted to determine the minimum set of shared libraries necessary for all
  the binaries I was interested in. The problem with this approach is that it
  requires that the binaries used the same exact set of shared libraries. I hit
  a bug where one binary depended on `libc-2.28.so` and another depended on
  `libc-2.29.so`. When the binary was run, it caused a stack-smashing error,
  similar to https://stackoverflow.com/questions/57156105.

  ```shell
  #!/bin/bash
  set -euo pipefail

  # Copies a binary and all required shared libraries into a new directory
  # preserving the directory structure.
  #
  # Usage:
  #     mirror-binary.sh <binary> <dest>
  #
  # NOTE: After copying the files, you should run `ldconfig` to regenerate symlinks
  # from the major to the minor versions of each shared library, e.g.:
  # libfoo.so.2 -> libfoo.so.2.1.2.
  #
  # Example:
  #     mirror-binary.sh /bin/bash /slim
  # Copies /bin/bash to /slim/bin/bash. Any shared libraries that /bin/bash
  # depends are also copied to /slim.

  if [[ $# -ne 2 ]]; then
    echo "ERROR: expected one binary and one destination in args: $@."
    exit 1
  fi

  binary="${1}"
  prefix="${2}"

  shared_libs=( $(ldd "${binary}" | grep -Eoh '(/usr)?/lib/[^ ]+' | xargs   --no-run-if-empty -n1 readlink -f) )
  for lib in "${shared_libs[@]}"; do
    if [[ "${lib}" == */libc*.so ]]; then
      # Skip libc because it should be included on the base image.
      continue;
    fi
    mirror-file.sh "${lib}" "${prefix}"
  done

  mirror-file.sh "${binary}" "${prefix}"
  ```

  ```shell
  #!/bin/bash
  set -euo pipefail

  # Copies a file into a new directory preserving the directory structure.
  #
  # Usage:
  #
  #     mirror-file.sh <file> <dest>
  #
  # Example:
  #
  #     mirror-file.sh /etc/passwd /slim
  #
  # Copies /etc/passwd to /slim/etc/passwd.

  if [[ $# -ne 2 ]]; then
    echo "ERROR: expected one file and one destination in args: $@."
    exit 1
  fi

  file="${1}"
  prefix="${2}"

  if [[ ! -f "${file}" ]]; then
    echo "Expected ${file} to be a file"
    exit 1
  fi

  dest="$(dirname ${prefix}${file})"
  mkdir -p "${dest}"
  cp -f "${file}" "${dest}"
  ```
