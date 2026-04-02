APP      := witti
API_APP  := witti-api
WEB_APP  := witti-web

MAKEFLAGS += -j4

ifeq ($(OS),Windows_NT)
  EXE        := .exe
  BIN        := $(APP)$(EXE)
  API_BIN    := $(API_APP)$(EXE)
  WEB_BIN    := $(WEB_APP)$(EXE)
  RM_BIN     := powershell -NoProfile -Command "if (Test-Path './$(BIN)') { Remove-Item './$(BIN)' }"
  RM_API_BIN := powershell -NoProfile -Command "if (Test-Path './$(API_BIN)') { Remove-Item './$(API_BIN)' }"
  RM_WEB_BIN := powershell -NoProfile -Command "if (Test-Path './$(WEB_BIN)') { Remove-Item './$(WEB_BIN)' }"
  RM_BIN_DIR := powershell -NoProfile -Command "if (Test-Path './bin') { Remove-Item -Recurse -Force './bin' }"
  MKDIR_BIN  := cmd /c "if not exist bin mkdir bin"
  # Windows: env vars must be set via 'set' in cmd.exe before go build
  define BUILD_ONE
  cmd /c "set GOOS=$(1)&& set GOARCH=$(2)&& go build -o bin/$(APP)-$(1)-$(2)$(3) ./cmd/witti" && echo   built bin/$(APP)-$(1)-$(2)$(3)
  endef
  define BUILD_ONE_API
  cmd /c "set GOOS=$(1)&& set GOARCH=$(2)&& go build -o bin/$(API_APP)-$(1)-$(2)$(3) ./cmd/witti-api" && echo   built bin/$(API_APP)-$(1)-$(2)$(3)
  endef
  define BUILD_ONE_WEB
  cmd /c "set GOOS=$(1)&& set GOARCH=$(2)&& go build -o bin/$(WEB_APP)-$(1)-$(2)$(3) ./cmd/witti-web" && echo   built bin/$(WEB_APP)-$(1)-$(2)$(3)
  endef
else
  EXE        :=
  BIN        := $(APP)
  API_BIN    := $(API_APP)
  WEB_BIN    := $(WEB_APP)
  RM_BIN     := rm -f ./$(BIN)
  RM_API_BIN := rm -f ./$(API_BIN)
  RM_WEB_BIN := rm -f ./$(WEB_BIN)
  RM_BIN_DIR := rm -rf ./bin
  MKDIR_BIN  := mkdir -p bin
  # Unix: inline env-var assignment
  define BUILD_ONE
  GOOS=$(1) GOARCH=$(2) go build -o bin/$(APP)-$(1)-$(2)$(3) ./cmd/witti && echo "  built bin/$(APP)-$(1)-$(2)$(3)"
  endef
  define BUILD_ONE_API
  GOOS=$(1) GOARCH=$(2) go build -o bin/$(API_APP)-$(1)-$(2)$(3) ./cmd/witti-api && echo "  built bin/$(API_APP)-$(1)-$(2)$(3)"
  endef
  define BUILD_ONE_WEB
  GOOS=$(1) GOARCH=$(2) go build -o bin/$(WEB_APP)-$(1)-$(2)$(3) ./cmd/witti-web && echo "  built bin/$(WEB_APP)-$(1)-$(2)$(3)"
  endef
endif

.PHONY: help build build-api build-web build-all run run-api run-web test test-coverage test-verbose test-all clean

help:
	@echo "Targets:"
	@echo "  build          Build $(BIN) for the current OS/arch"
	@echo "  build-api      Build $(API_BIN) for the current OS/arch"
	@echo "  build-web      Build $(WEB_BIN) for the current OS/arch"
	@echo "  build-all      Cross-compile all binaries for all platforms into ./bin/"
	@echo "  run            Run the CLI (pass args with ARGS='tokyo -limit 5')"
	@echo "  run-api        Run the REST API (pass args with ARGS='-addr :8080')"
	@echo "  run-web        Run the web UI + API server (pass args with ARGS='-addr :8080')"
	@echo "  test           Run go test ./..."
	@echo "  test-coverage  Run tests and print coverage summary"
	@echo "  test-verbose   Run tests verbosely"
	@echo "  test-all       Run verbose and coverage tests"
	@echo "  clean          Remove built executables"

build:
	go build -o "$(BIN)" ./cmd/witti

build-api:
	go build -o "$(API_BIN)" ./cmd/witti-api

build-web:
	go build -o "$(WEB_BIN)" ./cmd/witti-web

build-all:
	$(MKDIR_BIN)
	$(call BUILD_ONE,linux,amd64,)
	$(call BUILD_ONE,linux,arm64,)
	$(call BUILD_ONE,darwin,amd64,)
	$(call BUILD_ONE,darwin,arm64,)
	$(call BUILD_ONE,windows,amd64,.exe)
	$(call BUILD_ONE,windows,arm64,.exe)
	$(call BUILD_ONE,freebsd,amd64,)
	$(call BUILD_ONE_API,linux,amd64,)
	$(call BUILD_ONE_API,linux,arm64,)
	$(call BUILD_ONE_API,darwin,amd64,)
	$(call BUILD_ONE_API,darwin,arm64,)
	$(call BUILD_ONE_API,windows,amd64,.exe)
	$(call BUILD_ONE_API,windows,arm64,.exe)
	$(call BUILD_ONE_API,freebsd,amd64,)
	$(call BUILD_ONE_WEB,linux,amd64,)
	$(call BUILD_ONE_WEB,linux,arm64,)
	$(call BUILD_ONE_WEB,darwin,amd64,)
	$(call BUILD_ONE_WEB,darwin,arm64,)
	$(call BUILD_ONE_WEB,windows,amd64,.exe)
	$(call BUILD_ONE_WEB,windows,arm64,.exe)
	$(call BUILD_ONE_WEB,freebsd,amd64,)

run:
	go run ./cmd/witti $(ARGS)

run-api:
	go run ./cmd/witti-api $(ARGS)

run-web:
	go run ./cmd/witti-web $(ARGS)

test:
	go test ./...

test-verbose:
	go test -v ./...

test-coverage:
	go test -coverprofile=cover.out -count=1 ./...
	go tool cover -func=cover.out

test-all: test-verbose test-coverage

clean:
	$(RM_BIN)
	$(RM_API_BIN)
	$(RM_WEB_BIN)
	$(RM_BIN_DIR)
