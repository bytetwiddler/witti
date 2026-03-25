APP := witti

MAKEFLAGS += -j4

ifeq ($(OS),Windows_NT)
  EXE      := .exe
  BIN      := $(APP)$(EXE)
  RM_BIN   := powershell -NoProfile -Command "if (Test-Path './$(BIN)') { Remove-Item './$(BIN)' }"
  RM_BIN_DIR := powershell -NoProfile -Command "if (Test-Path './bin') { Remove-Item -Recurse -Force './bin' }"
  MKDIR_BIN  := cmd /c "if not exist bin mkdir bin"
  # Windows: env vars must be set via 'set' in cmd.exe before go build
  define BUILD_ONE
  cmd /c "set GOOS=$(1)&& set GOARCH=$(2)&& go build -o bin/$(APP)-$(1)-$(2)$(3) ./cmd/witti" && echo   built bin/$(APP)-$(1)-$(2)$(3)
  endef
else
  EXE      :=
  BIN      := $(APP)
  RM_BIN   := rm -f ./$(BIN)
  RM_BIN_DIR := rm -rf ./bin
  MKDIR_BIN  := mkdir -p bin
  # Unix: inline env-var assignment
  define BUILD_ONE
  GOOS=$(1) GOARCH=$(2) go build -o bin/$(APP)-$(1)-$(2)$(3) ./cmd/witti && echo "  built bin/$(APP)-$(1)-$(2)$(3)"
  endef
endif

.PHONY: help build build-all run test test-coverage clean

help:
	@echo "Targets:"
	@echo "  build      Build $(BIN) for the current OS/arch"
	@echo "  build-all  Cross-compile for all platforms into ./bin/"
	@echo "  run        Run the app (pass args with ARGS='tokyo -limit 5')"
	@echo "  test       Run go test ./..."
	@echo "  test-coverage  Run tests and print coverage summary"
	@echo "  test-verbose  Run tests and print coverage summary"
	@echo "  test-all  Run verbose and coverage tests"
	@echo "  clean      Remove built executables"

build:
	go build -o "$(BIN)" ./cmd/witti

build-all:
	$(MKDIR_BIN)
	$(call BUILD_ONE,linux,amd64,)
	$(call BUILD_ONE,linux,arm64,)
	$(call BUILD_ONE,darwin,amd64,)
	$(call BUILD_ONE,darwin,arm64,)
	$(call BUILD_ONE,windows,amd64,.exe)
	$(call BUILD_ONE,windows,arm64,.exe)
	$(call BUILD_ONE,freebsd,amd64,)

run:
	go run ./cmd/witti $(ARGS)

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
	$(RM_BIN_DIR)
