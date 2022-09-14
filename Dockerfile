FROM golang:1.19.1-alpine3.16 AS builder

ARG CORERULESET_VERSION=v4.0.0-rc1
ARG CORERULESET_MD5=9140236dc7e941c274e414385824c996

WORKDIR /app

RUN set -eux; \
    apk add --no-cache \
        gc-dev \
        make \
        gcc \
        git \
        musl-dev \
        pcre-dev \
        wget \
    && wget -qO/tmp/coreruleset.tar.gz https://github.com/coreruleset/coreruleset/archive/${CORERULESET_VERSION}.tar.gz \
    && echo "$CORERULESET_MD5  /tmp/coreruleset.tar.gz" | md5sum -c \
    && mkdir -p /tmp/coraza-coreruleset \
    && mkdir -p /etc/coraza-spoa/rules \
    && tar xzf /tmp/coreruleset.tar.gz --strip-components=1 -C /tmp/coraza-coreruleset \
    && rm /tmp/coreruleset.tar.gz \
    && cp /tmp/coraza-coreruleset/crs-setup.conf.example /etc/coraza-spoa/crs-setup.conf \
    && cp /tmp/coraza-coreruleset/rules/* /etc/coraza-spoa/rules/ \
    && rm -rf /tmp/coraza-coreruleset \
    && find /etc/coraza-spoa/rules -type f -name '*.example' | while read -r f; do cp -p "$f" "${f%.example}"; done \
    && sed -i.example 's/^\(SecDefaultAction "phase:[12]\),log,auditlog,pass"/\1,log,noauditlog,deny,status:403"/' /etc/coraza-spoa/crs-setup.conf

COPY . .

RUN make

RUN mkdir -p /app/start \
    && mv /app/docker/coraza-spoa/plugins /etc/coraza-spoa/plugins \
    && mv /app/docker/coraza-spoa/coraza.conf /etc/coraza-spoa/coraza.conf \
    && mv /app/config.yaml.default /app/start/config.yaml \
    && mv /app/coraza-spoa_amd64 /app/start/coraza-spoa \
    && mv /app/docker/start.sh /app/start/start.sh \
    && chmod +x /app/start/start.sh \
    && chmod +x /app/start/coraza-spoa



FROM alpine:3.16 AS production

RUN apk add --no-cache \
        tini \
        pcre-dev \
    && mkdir -p /var/log/coraza-spoa

COPY --from=builder /app/start /
COPY --from=builder /etc/coraza-spoa /etc/coraza-spoa

ENTRYPOINT ["tini", "--", "/start.sh"]
