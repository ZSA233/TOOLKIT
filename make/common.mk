SHELL := /bin/bash

GO ?= go
PNPM ?= pnpm
UV ?= uv

# Prefer the selected Go binary/toolchain over any stale shell-level GOROOT.
unexport GOROOT

GO_MODULE_TEST_CMD = cd $(1) && $(GO) test $(2) ./...
GO_MODULE_TIDY_CMD = cd $(1) && $(GO) mod tidy $(2)

.DEFAULT_GOAL := help
