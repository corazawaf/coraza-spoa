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

.DEFAULT_GOAL := build

HOST_ARCH = $(shell which go >/dev/null 2>&1 && go env GOARCH)
ARCH ?= $(HOST_ARCH)
ifeq ($(ARCH),)
    $(error mandatory variable ARCH is empty, either set it when calling the command or make sure 'go env GOARCH' works)
endif

HOST_OS = $(shell which go >/dev/null 2>&1 && go env GOOS)
OS ?= $(HOST_OS)
ifeq ($(OS),)
	$(error mandatory variable OS is empty, either set it when calling the cammand or make sure 'go env GOOS' works)
endif

EXECUTABLE_FILE = coraza-server
CONFIGURATION_FILE = config.yaml

.PHONY: build
build:
	@GOARCH=$(ARCH) go build -o $(EXECUTABLE_FILE) cmd/main.go
	@cp config.yaml.default $(CONFIGURATION_FILE)

.PHONY: clean
clean: $(BUILD_FILES)
	@rm -rf $(EXECUTABLE_FILE)
	@rm -rf $(CONFIGURATION_FILE)


