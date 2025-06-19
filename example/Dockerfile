# Copyright 2023 The OWASP Coraza contributors
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.23@sha256:dd5cc4b4f85d13329cb5b17cbf35c509e1c82a43bf6e5961516fda444013121a AS build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN go vet -v ./...

RUN CGO_ENABLED=0 go build -o /go/bin/coraza-spoa

FROM gcr.io/distroless/static-debian11@sha256:1dbe426d60caed5d19597532a2d74c8056cd7b1674042b88f7328690b5ead8ed

LABEL org.opencontainers.image.authors="The OWASP Coraza contributors" \
      org.opencontainers.image.description="OWASP Coraza WAF (Haproxy SPOA)" \
      org.opencontainers.image.documentation="https://coraza.io/connectors/coraza-spoa/" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.source="https://github.com/corazawaf/coraza-spoa" \
      org.opencontainers.image.title="coraza-spoa"

COPY --from=build /go/bin/coraza-spoa /
COPY ./example/coraza-spoa.yaml /config.yaml

CMD ["/coraza-spoa", "--config", "/config.yaml"]