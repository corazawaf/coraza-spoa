# Copyright 2023 The OWASP Coraza contributors
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.25@sha256:9006890ecba0a168034d99516084099ae3114d9f2b7d6572c77f2dde57ebc980 AS build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN go vet -v ./...

RUN CGO_ENABLED=0 go build -o /go/bin/coraza-spoa

FROM gcr.io/distroless/static-debian13@sha256:f2ea2709ac8db56323cbd7d014277f32cb572d9ea124b0076f7aafe5980678fe

LABEL org.opencontainers.image.authors="The OWASP Coraza contributors" \
      org.opencontainers.image.description="OWASP Coraza WAF (Haproxy SPOA)" \
      org.opencontainers.image.documentation="https://coraza.io/connectors/coraza-spoa/" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.source="https://github.com/corazawaf/coraza-spoa" \
      org.opencontainers.image.title="coraza-spoa"

COPY --from=build /go/bin/coraza-spoa /
COPY ./example/coraza-spoa.yaml /config.yaml

CMD ["/coraza-spoa", "--config", "/config.yaml"]
