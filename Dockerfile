FROM ubuntu:24.04

WORKDIR /src

# Install some system dependencies
RUN apt-get update && apt-get install -y xz-utils curl git ca-certificates
RUN update-ca-certificates

# This is the official way to install it :(
RUN curl -fsSL https://get.jetify.com/devbox -o install-devbox.sh
RUN chmod +x install-devbox.sh
RUN ./install-devbox.sh -f

# Do just dependencies first, to benefit from layer caching
COPY devbox.json devbox.json
COPY devbox.lock devbox.lock

# This also installs nix, takes a while, todo, do this earlier in Dockerfile
RUN devbox install

# Now the rest!
COPY . .
RUN devbox run build

# And copy the command to somewhere we can find it
RUN mv /src/out/ecstatic /usr/bin/ecstatic
