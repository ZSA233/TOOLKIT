GO_WORK_MODULES := services/mtu_tuner
GO_WORK_TEST_TARGETS := mtu-tuner-test
GO_WORK_TIDY_TARGETS := mtu-tuner-tidy

LIBS_APPKIT_DIR := libs/appkit
LIBS_CLIENTS_DIR := libs/clients
LIBS_UTILS_DIR := libs/utils
GO_WORK_TEST_FLAGS ?= -short
GO_WORK_TIDY_FLAGS ?=


.PHONY: go-work-test
go-work-test: $(GO_WORK_TEST_TARGETS)

.PHONY: go-work-tidy
go-work-tidy: $(GO_WORK_TIDY_TARGETS)

.PHONY: go-work-check
go-work-check: go-work-tidy go-work-test

.PHONY: check
check: go-work-check
