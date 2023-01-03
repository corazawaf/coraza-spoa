FROM golang:1.19.1-alpine3.16 AS build

# Specify Coreruleset version to download
ARG CORERULESET_VERSION=v4.0.0-rc1
ARG CORERULESET_MD5=9140236dc7e941c274e414385824c996

# Change working directory
WORKDIR /home/coraza-spoa

RUN \
    apk add --no-cache \
        # Install make to build coraza-spoa binary from makefile
        make \
    # Download and set up Coreruleset
    && wget -qO/tmp/coreruleset.tar.gz https://github.com/coreruleset/coreruleset/archive/${CORERULESET_VERSION}.tar.gz \
    && echo "$CORERULESET_MD5  /tmp/coreruleset.tar.gz" | md5sum -c \
    && mkdir -p /tmp/coraza-coreruleset \
    && mkdir -p /etc/coraza-spoa/rules \
    && tar xzf /tmp/coreruleset.tar.gz --strip-components=1 -C /tmp/coraza-coreruleset \
    && mv /tmp/coraza-coreruleset/crs-setup.conf.example /etc/coraza-spoa/crs-setup.conf \
    && mv /tmp/coraza-coreruleset/rules /etc/coraza-spoa \
    && mv /tmp/coraza-coreruleset/plugins /etc/coraza-spoa \
    && rm -rf /tmp/*

# Cache Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy project files into build image
COPY . .

# Build coraza-spoa binary
RUN make

# --------------------------

FROM alpine:3.16
RUN addgroup -g 1001 coraza-spoa \
 && adduser -u 1001 -G coraza-spoa -D -s /bin/false coraza-spoa \
 && mkdir -p /var/log/coraza-spoa \
 && chown coraza-spoa:coraza-spoa /var/log/coraza-spoa

# Copy Coreruleset files and binary from build image
COPY --from=build /etc/coraza-spoa /etc/coraza-spoa
COPY --from=build /home/coraza-spoa/docker/coraza-spoa/coraza.conf /etc/coraza-spoa/coraza.conf
COPY --from=build /home/coraza-spoa/config.yaml.default /home/coraza-spoa/config.yaml
COPY --from=build /home/coraza-spoa/coraza-spoa_amd64 /usr/local/bin/coraza-spoa
# Make binary executable and change the owner of files in the home folder
RUN chmod +x /usr/local/bin/coraza-spoa && chown -R coraza-spoa:coraza-spoa /home/coraza-spoa

USER coraza-spoa
WORKDIR /home/coraza-spoa/

CMD ["coraza-spoa", "-config", "config.yaml"]
