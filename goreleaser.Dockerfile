FROM debian:buster-slim

ARG TARGETPLATFORM


WORKDIR /app

COPY . .

RUN set -ex \
  && mv neutrinod /usr/local/bin/neutrinod \
  && mv neutrino /usr/local/bin/neutrino


RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates

ENV NEUTRINO_ELEMENTS_DB_MIGRATION_PATH="file://"

# expose websockets & webhook port
EXPOSE 8000

ENTRYPOINT ["neutrinod"]
