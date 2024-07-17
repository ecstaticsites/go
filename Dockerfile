############################
# STAGE 0 - Build
############################

# https://hub.docker.com/r/jetpackio/devbox/tags
FROM jetpackio/devbox:0.12.0

WORKDIR /code

USER root:root
RUN mkdir -p /code && chown ${DEVBOX_USER}:${DEVBOX_USER} /code
USER ${DEVBOX_USER}:${DEVBOX_USER}
COPY --chown=${DEVBOX_USER}:${DEVBOX_USER} devbox.json devbox.json
COPY --chown=${DEVBOX_USER}:${DEVBOX_USER} devbox.lock devbox.lock

# Do just dependencies first, to benefit from layer caching

RUN devbox install

# Now the rest!

COPY --chown=${DEVBOX_USER}:${DEVBOX_USER} . .

RUN devbox run build

############################
# STAGE 1 - Run
############################

# https://hub.docker.com/_/ubuntu/tags
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y git ca-certificates lftp
RUN update-ca-certificates

COPY --from=0 /code/out/cbnr /usr/bin/cbnr
