FROM golang:1.18.4-alpine3.16 AS builder

WORKDIR /app

COPY . .

RUN set -eux; \
    apk add --no-cache \
        gc-dev \
        gcc \
        git \
        musl-dev \
        pcre-dev \
    ;

RUN CGO_ENABLED=1 GOARCH=amd64 GOOS=linux go build -o coraza-spoa cmd/main.go

FROM alpine:3.16 AS production

ARG CORERULESET_VERSION=v3.3.2
ARG CORERULESET_MD5=244db30b807c56d8277010b70305efda

COPY ./build/coraza.conf /etc/coraza-spoa/coraza.conf
COPY ./build/start.sh ./build/config.yml ./
COPY --from=builder /app/coraza-spoa ./coraza-spoa

RUN set -eux; \
    apk add --no-cache \
        tini \
        wget \
        pcre-dev \
    && chmod +x /start.sh \
    && wget -qO/tmp/coreruleset.tar.gz https://github.com/coreruleset/coreruleset/archive/${CORERULESET_VERSION}.tar.gz \
    && echo "$CORERULESET_MD5  /tmp/coreruleset.tar.gz" | md5sum -c \
    && mkdir -p /tmp/coraza-coreruleset \
    && mkdir -p /etc/coraza-spoa/rules \
    && tar xzf /tmp/coreruleset.tar.gz --strip-components=1 -C /tmp/coraza-coreruleset \
    && rm /tmp/coreruleset.tar.gz \
    && cp /tmp/coraza-coreruleset/crs-setup.conf.example /etc/coraza-spoa/crs-setup.conf \
    && cp /tmp/coraza-coreruleset/rules/* /etc/coraza-spoa/rules/ \
    && rm -rf /tmp/coraza-coreruleset \
    && mkdir -p /var/log/coraza-spoa \
    && touch /var/log/coraza-spoa/error.log \
    && ln -sf /proc/1/fd/1 /var/log/coraza-spoa/error.log

ENTRYPOINT ["tini", "--", "/start.sh"]
