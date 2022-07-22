FROM debian:buster-slim

ARG TARGETPLATFORM


WORKDIR /app

COPY . .

RUN set -ex \
  && if [ "${TARGETPLATFORM}" = "linux/amd64" ]; then export TARGETPLATFORM=amd64; fi \
  && if [ "${TARGETPLATFORM}" = "linux/arm64" ]; then export TARGETPLATFORM=arm64; fi \
  && mv neutrinod /usr/local/bin/neutrinod \
  && mv "neutrinod-linux-$TARGETPLATFORM" /usr/local/bin/neutrinod


RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates

ENV NEUTRINO_ELEMENTS_DB_MIGRATION_PATH="file://"

# expose trader and operator interface ports
EXPOSE 8000

ENTRYPOINT ["neutrinod"]
