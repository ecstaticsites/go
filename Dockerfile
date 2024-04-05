# Oh yeah, ubuntu with docker running. Needed for toast to work for building
# sites on git push, but much MUCH better would be to fork toast and have it
# submit its containers as jobs to the k8s API...
FROM cruizba/ubuntu-dind:noble-26.0.0

WORKDIR /app

RUN apt-get update && apt-get install -y git ca-certificates git-ftp
RUN update-ca-certificates

ARG VERSION=0.47.6
RUN curl https://raw.githubusercontent.com/stepchowfun/toast/main/install.sh -LSfs | sh

COPY misc/GeoLite2-Country.mmdb .

COPY out/cbnr /usr/bin/cbnr

# need to use entrypoint from base image, it's what starts docker
ENTRYPOINT ["entrypoint.sh"]
