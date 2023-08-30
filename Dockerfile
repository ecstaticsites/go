FROM ubuntu:22.04

WORKDIR /app

COPY GeoLite2-Country.mmdb .

COPY out/cbnr .

ENTRYPOINT ["./cbnr"]
