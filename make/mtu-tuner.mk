MTU_TUNER_DIR := services/mtu_tuner
MTU_TUNER_CMD_DIR := $(MTU_TUNER_DIR)/cmd
MTU_TUNER_GUI_DIR := $(MTU_TUNER_CMD_DIR)/gui
MTU_TUNER_GUI_FRONTEND_DIR := $(MTU_TUNER_GUI_DIR)/frontend
MTU_TUNER_SCRIPTS_DIR := $(MTU_TUNER_DIR)/scripts
MTU_TUNER_BLUEPRINT_CONFIG ?= $(MTU_TUNER_SCRIPTS_DIR)/api-blueprint.toml
API_BLUEPRINT_RUN ?= $(if $(strip $(API_BLUEPRINT_PROJECT)),$(UV) run --project $(API_BLUEPRINT_PROJECT),)

MTU_TUNER_TARGETS := cli
MTU_TUNER_GUI_TARGETS := gui
MTU_TUNER_ALL_TARGETS := $(MTU_TUNER_TARGETS) $(MTU_TUNER_GUI_TARGETS)

CMD ?= cli
BIN_NAME ?= $(CMD)
HOST_GOOS := $(shell $(GO) env GOOS)
GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)
CGO_ENABLED ?= 0
GO_BUILD_FLAGS ?=
LD_FLAGS ?=
GO_TEST_FLAGS ?= -short
# Keep mtu_tuner builds pinned to its own go.mod so copied checkouts do not
# inherit unrelated repo-level go.work entries that may be absent locally.
MTU_TUNER_GO_ENV := GOWORK=off
MTU_TUNER_BIN_ROOT := $(MTU_TUNER_DIR)/build/bin
MTU_TUNER_PACKAGE_ROOT := $(MTU_TUNER_DIR)/build/packages
MTU_TUNER_BIN_DIR = $(MTU_TUNER_BIN_ROOT)/$(1)/$(GOOS)_$(GOARCH)
MTU_TUNER_MAIN_PKG = ./cmd/$(1)
MTU_TUNER_BIN_NAME = $(if $(filter %.exe,$(or $(2),$(1))),$(or $(2),$(1)),$(or $(2),$(1))$(if $(filter windows,$(GOOS)),.exe,))
MTU_TUNER_BIN_PATH = $(call MTU_TUNER_BIN_DIR,$(1))/$(call MTU_TUNER_BIN_NAME,$(1),$(2))
MTU_TUNER_BIN_ABS_PATH = $(abspath $(call MTU_TUNER_BIN_PATH,$(1),$(2)))
MTU_TUNER_GUI_TSC_BUILDINFO := $(MTU_TUNER_GUI_FRONTEND_DIR)/node_modules/.tmp/tsconfig.app.tsbuildinfo
MTU_TUNER_GUI_OUTPUT_DIR = $(call MTU_TUNER_BIN_DIR,gui)
GUI_BIN_NAME ?=
MTU_TUNER_GUI_BIN_NAME ?= $(if $(strip $(GUI_BIN_NAME)),$(GUI_BIN_NAME),gui)
MTU_TUNER_GUI_BIN_PATH := $(call MTU_TUNER_BIN_PATH,gui,$(MTU_TUNER_GUI_BIN_NAME))
MTU_TUNER_GUI_BIN_ABS_PATH := $(abspath $(MTU_TUNER_GUI_BIN_PATH))
MTU_TUNER_GUI_BUILD_TAGS ?= mtu_tuner_embed_frontend
WINDOWS_GOARCH ?= amd64
WINDOWS_CC ?=
WINDOWS_CXX ?=
MTU_TUNER_PACKAGE_NAME ?= mtu-tuner
MTU_TUNER_WINDOWS_GUI_BIN_DIR := $(MTU_TUNER_BIN_ROOT)/gui/windows_$(WINDOWS_GOARCH)
MTU_TUNER_WINDOWS_GUI_BIN_BASE := $(if $(filter %.exe,$(MTU_TUNER_GUI_BIN_NAME)),$(MTU_TUNER_GUI_BIN_NAME),$(MTU_TUNER_GUI_BIN_NAME).exe)
MTU_TUNER_WINDOWS_GUI_BIN_PATH := $(MTU_TUNER_WINDOWS_GUI_BIN_DIR)/$(MTU_TUNER_WINDOWS_GUI_BIN_BASE)
MTU_TUNER_WINDOWS_PACKAGE_DIR := $(MTU_TUNER_PACKAGE_ROOT)/gui/windows_$(WINDOWS_GOARCH)/$(MTU_TUNER_PACKAGE_NAME)
MTU_TUNER_WINDOWS_PACKAGE_ZIP := $(MTU_TUNER_PACKAGE_ROOT)/gui/windows_$(WINDOWS_GOARCH)/$(MTU_TUNER_PACKAGE_NAME)_windows_$(WINDOWS_GOARCH).zip
# Windows GUI binaries should use the GUI subsystem so launching the app does not open an extra console window.
MTU_TUNER_GUI_LD_FLAGS = $(strip $(LD_FLAGS) $(if $(filter windows,$(GOOS)),-H=windowsgui,))

MTU_TUNER_BLUEPRINT_CMD = cd $(dir $(MTU_TUNER_BLUEPRINT_CONFIG)) &&
MTU_TUNER_BLUEPRINT_CONFIG_ARG = -c $(notdir $(MTU_TUNER_BLUEPRINT_CONFIG))
MTU_TUNER_API_CHECK_CMD = $(MTU_TUNER_BLUEPRINT_CMD) $(API_BLUEPRINT_RUN) api-gen check $(MTU_TUNER_BLUEPRINT_CONFIG_ARG)
MTU_TUNER_API_GEN_GOLANG_CMD = $(MTU_TUNER_BLUEPRINT_CMD) $(API_BLUEPRINT_RUN) api-gen generate $(MTU_TUNER_BLUEPRINT_CONFIG_ARG) --target go.server
MTU_TUNER_API_GEN_TYPESCRIPT_CMD = $(MTU_TUNER_BLUEPRINT_CMD) $(API_BLUEPRINT_RUN) api-gen generate $(MTU_TUNER_BLUEPRINT_CONFIG_ARG) --target typescript.client
MTU_TUNER_API_GEN_WAILS_CMD = $(MTU_TUNER_BLUEPRINT_CMD) GOWORK=off $(API_BLUEPRINT_RUN) api-gen generate $(MTU_TUNER_BLUEPRINT_CONFIG_ARG) --target wails.v3
MTU_TUNER_API_GEN_ALL_CMD = $(MTU_TUNER_BLUEPRINT_CMD) GOWORK=off $(API_BLUEPRINT_RUN) api-gen generate $(MTU_TUNER_BLUEPRINT_CONFIG_ARG)

define mtu_tuner_build_target
.PHONY: mtu-tuner-build-$(1)
mtu-tuner-build-$(1):
	@echo "-> building $(1) ($(GOOS)/$(GOARCH))"
	@mkdir -p $(call MTU_TUNER_BIN_DIR,$(1))
	cd $(MTU_TUNER_DIR) && \
		$(MTU_TUNER_GO_ENV) CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) CC="$(CC)" CXX="$(CXX)" \
		$(GO) build $(GO_BUILD_FLAGS) -ldflags '$(LD_FLAGS)' \
		-o $(call MTU_TUNER_BIN_ABS_PATH,$(1),$(1)) $(call MTU_TUNER_MAIN_PKG,$(1))
endef
$(foreach target,$(MTU_TUNER_TARGETS),$(eval $(call mtu_tuner_build_target,$(target))))

.PHONY: mtu-tuner-build
mtu-tuner-build:
	@echo "-> building $(CMD) ($(GOOS)/$(GOARCH))"
	@mkdir -p $(call MTU_TUNER_BIN_DIR,$(CMD))
	cd $(MTU_TUNER_DIR) && \
		$(MTU_TUNER_GO_ENV) CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) CC="$(CC)" CXX="$(CXX)" \
		$(GO) build $(GO_BUILD_FLAGS) -ldflags '$(LD_FLAGS)' \
		-o $(call MTU_TUNER_BIN_ABS_PATH,$(CMD),$(BIN_NAME)) $(call MTU_TUNER_MAIN_PKG,$(CMD))

.PHONY: mtu-tuner-build-all
mtu-tuner-build-all: $(addprefix mtu-tuner-build-,$(MTU_TUNER_TARGETS))

.PHONY: mtu-tuner-run
mtu-tuner-run:
	cd $(MTU_TUNER_DIR) && $(MTU_TUNER_GO_ENV) $(GO) run ./cmd/$(CMD)

.PHONY: mtu-tuner-gui-frontend-build
mtu-tuner-gui-frontend-build: mtu-tuner-gui-frontend-install
	cd $(MTU_TUNER_GUI_FRONTEND_DIR) && $(PNPM) build

.PHONY: mtu-tuner-gui-frontend-install
mtu-tuner-gui-frontend-install:
	# Build-script approvals live in pnpm-workspace.yaml so CI installs stay non-interactive.
	cd $(MTU_TUNER_GUI_FRONTEND_DIR) && $(PNPM) install --frozen-lockfile

.PHONY: mtu-tuner-gui-build
mtu-tuner-gui-build: mtu-tuner-gui-frontend-build
	@echo "-> building gui ($(GOOS)/$(GOARCH))"
	@mkdir -p $(MTU_TUNER_GUI_OUTPUT_DIR)
	cd $(MTU_TUNER_DIR) && \
		$(MTU_TUNER_GO_ENV) CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) CC="$(CC)" CXX="$(CXX)" \
		$(GO) build $(GO_BUILD_FLAGS) -tags '$(MTU_TUNER_GUI_BUILD_TAGS)' -ldflags '$(MTU_TUNER_GUI_LD_FLAGS)' \
		-o $(MTU_TUNER_GUI_BIN_ABS_PATH) ./cmd/gui

.PHONY: mtu-tuner-gui-build-windows
mtu-tuner-gui-build-windows:
	@if [ "$(HOST_GOOS)" != "windows" ] && [ -z "$(WINDOWS_CC)" ]; then \
		echo "WINDOWS_CC is required when cross-building the GUI from $(HOST_GOOS)."; \
		echo "example: make mtu-tuner-gui-package-windows WINDOWS_GOARCH=$(WINDOWS_GOARCH) WINDOWS_CC='zig cc -target x86_64-windows-gnu' WINDOWS_CXX='zig c++ -target x86_64-windows-gnu'"; \
		exit 1; \
	fi
	$(MAKE) mtu-tuner-gui-build GOOS=windows GOARCH=$(WINDOWS_GOARCH) $(if $(WINDOWS_CC),CC='$(WINDOWS_CC)') $(if $(WINDOWS_CXX),CXX='$(WINDOWS_CXX)')

.PHONY: mtu-tuner-gui-package-windows
mtu-tuner-gui-package-windows: mtu-tuner-gui-build-windows
	@echo "-> packaging gui (windows/$(WINDOWS_GOARCH))"
	@rm -rf $(MTU_TUNER_WINDOWS_PACKAGE_DIR)
	@mkdir -p $(MTU_TUNER_WINDOWS_PACKAGE_DIR)
	cp $(MTU_TUNER_WINDOWS_GUI_BIN_PATH) $(MTU_TUNER_WINDOWS_PACKAGE_DIR)/$(MTU_TUNER_PACKAGE_NAME).exe
	@if command -v zip >/dev/null 2>&1; then \
		rm -f $(MTU_TUNER_WINDOWS_PACKAGE_ZIP); \
		cd $(dir $(MTU_TUNER_WINDOWS_PACKAGE_DIR)) && zip -qr $(notdir $(MTU_TUNER_WINDOWS_PACKAGE_ZIP)) $(notdir $(MTU_TUNER_WINDOWS_PACKAGE_DIR)); \
		echo "-> packaged $(MTU_TUNER_WINDOWS_PACKAGE_ZIP)"; \
	else \
		echo "-> zip not found; package directory left at $(MTU_TUNER_WINDOWS_PACKAGE_DIR)"; \
	fi

.PHONY: mtu-tuner-gui-run
mtu-tuner-gui-run: mtu-tuner-gui-frontend-build
	@echo "-> running gui"
	cd $(MTU_TUNER_DIR) && \
		$(MTU_TUNER_GO_ENV) CGO_ENABLED=1 $(GO) run ./cmd/gui

.PHONY: mtu-tuner-test
mtu-tuner-test:
	cd $(MTU_TUNER_DIR) && $(MTU_TUNER_GO_ENV) $(GO) test $(GO_TEST_FLAGS) ./...

.PHONY: mtu-tuner-tidy
mtu-tuner-tidy:
	cd $(MTU_TUNER_DIR) && $(MTU_TUNER_GO_ENV) $(GO) mod tidy

.PHONY: mtu-tuner-clean-ts-buildinfo
mtu-tuner-clean-ts-buildinfo:
	rm -f $(MTU_TUNER_GUI_TSC_BUILDINFO)

.PHONY: mtu-tuner-api-check
mtu-tuner-api-check:
	$(MTU_TUNER_API_CHECK_CMD)

.PHONY: mtu-tuner-api-gen-golang
mtu-tuner-api-gen-golang:
	$(MTU_TUNER_API_GEN_GOLANG_CMD)

.PHONY: mtu-tuner-api-gen-typescript
mtu-tuner-api-gen-typescript:
	$(MTU_TUNER_API_GEN_TYPESCRIPT_CMD)
	$(MAKE) mtu-tuner-clean-ts-buildinfo

.PHONY: mtu-tuner-api-gen-wails
mtu-tuner-api-gen-wails:
	$(MTU_TUNER_API_GEN_WAILS_CMD)
	$(MAKE) mtu-tuner-clean-ts-buildinfo

.PHONY: mtu-tuner-api-gen-all
mtu-tuner-api-gen-all:
	$(MTU_TUNER_API_GEN_ALL_CMD)
