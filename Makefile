# Copyright 2022 The Corazawaf Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

BINARY = coraza-spoa

VERSION ?= "dev"
REVISION ?= $(shell git rev-parse HEAD)

ARCH ?= $(shell which go >/dev/null 2>&1 && go env GOARCH)

ifeq ($(ARCH),)
	$(error mandatory variable ARCH is empty, either set it when calling the command or make sure 'go env GOARCH' works)
endif

OS ?= $(shell which go >/dev/null 2>&1 && go env GOOS)

ifeq ($(OS),)
	$(error mandatory variable OS is empty, either set it when calling the cammand or make sure 'go env GOOS' works)
endif

#LDFLAGS = -ldflags "-X main.Version=${VERSION} -X main.Revision=${REVISION}"


default: docker

build:
	GOARCH=$(ARCH) GOOS=$(OS) go build -v ${LDFLAGS} -o $(BINARY)_$(ARCH) cmd/main.go

clean:
	rm -f $(BINARY)_amd64 $(BINARY)_arm64 $(BINARY)_386
