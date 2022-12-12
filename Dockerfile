FROM golang:1.19.1-alpine3.16 AS build

# Specify Coreruleset version to download
ARG CORERULESET_VERSION=v4.0.0-rc1
ARG CORERULESET_MD5=9140236dc7e941c274e414385824c996

# Change working directory
WORKDIR /app

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

RUN \
    # Move coraza-spoa config file to config dir
    mv /app/docker/coraza-spoa/coraza.conf /etc/coraza-spoa/coraza.conf \
    # Rename coraza-spoa default config file
    && mv /app/config.yaml.default /app/config.yaml \
    # Rename coraza-spoa binary
    && mv /app/coraza-spoa_amd64 /app/coraza-spoa \
    # Make coraza-spoa binary executable
    && chmod +x /app/coraza-spoa



FROM alpine:3.16
# Make directory for coraza-spoa audit and error logs
RUN mkdir -p /var/log/coraza-spoa
# Copy coraza-spoa binary and default config file from build image
COPY --from=build /app/config.yaml /app/coraza-spoa /
# Copy Coreruleset files from build image
COPY --from=build /etc/coraza-spoa /etc/coraza-spoa

# Container run command
CMD ["/coraza-spoa", "-config", "/config.yaml"]
