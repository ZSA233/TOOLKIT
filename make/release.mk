PYTHON ?= python3
RELEASECTL := $(PYTHON) scripts/releasectl.py
TOOL ?=
WORKFLOW ?= ci
RC ?=
TAG ?=
TARGET ?=
BASE_VERSION ?=
CHECK ?=
RELEASE_TAG ?= $(TAG)
HOST_GOOS ?= $(shell $(GO) env GOOS)

define require_release_tool
	@if [ -z "$(TOOL)" ]; then \
		echo "TOOL is required, for example: make $(1) TOOL=mtu-tuner"; \
		exit 1; \
	fi
endef

define require_release_tag
	@if [ -z "$(RELEASE_TAG)" ]; then \
		echo "RELEASE_TAG is required, for example: make $(1) TOOL=mtu-tuner RELEASE_TAG=mtu-tuner/v0.0.2"; \
		exit 1; \
	fi
endef

.PHONY: release-validate
release-validate:
	$(call require_release_tool,release-validate)
	$(RELEASECTL) validate --tool $(TOOL) $(if $(WORKFLOW),--workflow $(WORKFLOW)) $(if $(TAG),--tag $(TAG))

.PHONY: release-version
release-version:
	$(call require_release_tool,release-version)
	$(RELEASECTL) version --tool $(TOOL)

.PHONY: release-version-show
release-version-show: release-version

.PHONY: release-tag
release-tag:
	$(call require_release_tool,release-tag)
	$(RELEASECTL) tag --tool $(TOOL) $(if $(RC),--rc $(RC))

.PHONY: release-version-stable
release-version-stable:
	$(call require_release_tool,release-version-stable)
	@version="$$( $(RELEASECTL) version --tool $(TOOL) )"; \
	if [ "$(CHECK)" = "1" ] && [ -n "$(BASE_VERSION)" ] && [ "$$version" != "$(BASE_VERSION)" ]; then \
		echo "release version $$version does not match BASE_VERSION $(BASE_VERSION)"; \
		exit 1; \
	fi; \
	$(RELEASECTL) tag --tool $(TOOL)

.PHONY: release-version-rc
release-version-rc:
	$(call require_release_tool,release-version-rc)
	@if [ -z "$(RC)" ]; then \
		echo "RC is required, for example: make release-version-rc TOOL=mtu-tuner BASE_VERSION=0.0.2 RC=1 CHECK=1"; \
		exit 1; \
	fi
	@version="$$( $(RELEASECTL) version --tool $(TOOL) )"; \
	if [ "$(CHECK)" = "1" ] && [ -n "$(BASE_VERSION)" ] && [ "$$version" != "$(BASE_VERSION)" ]; then \
		echo "release version $$version does not match BASE_VERSION $(BASE_VERSION)"; \
		exit 1; \
	fi; \
	$(RELEASECTL) tag --tool $(TOOL) --rc $(RC)

.PHONY: release-metadata
release-metadata:
	$(call require_release_tool,release-metadata)
	$(RELEASECTL) metadata --tool $(TOOL) $(if $(TAG),--tag $(TAG))

.PHONY: release-matrix
release-matrix:
	$(call require_release_tool,release-matrix)
	$(RELEASECTL) github-matrix --tool $(TOOL) --workflow $(WORKFLOW)

.PHONY: release-workflow-tools
release-workflow-tools:
	$(RELEASECTL) workflow-tools --workflow $(WORKFLOW)

.PHONY: release-workflow-matrix
release-workflow-matrix:
	$(RELEASECTL) workflow-matrix --workflow $(WORKFLOW)

.PHONY: release-tag-check
release-tag-check:
	$(call require_release_tool,release-tag-check)
	$(call require_release_tag,release-tag-check)
	@workflow="$$( $(RELEASECTL) workflow-from-tag --tag "$(RELEASE_TAG)" )"; \
	$(RELEASECTL) validate --tool $(TOOL) --workflow "$$workflow" --tag "$(RELEASE_TAG)"

.PHONY: release-package
release-package:
	$(call require_release_tool,release-package)
	@if [ -z "$(TARGET)" ]; then \
		echo "TARGET is required, for example: make release-package TOOL=mtu-tuner TARGET=gui-linux-amd64"; \
		exit 1; \
	fi
	$(RELEASECTL) package --tool $(TOOL) --target $(TARGET)

.PHONY: release-preflight
release-preflight:
	$(call require_release_tool,release-preflight)
	$(call require_release_tag,release-preflight)
	python3 -m unittest -v scripts.tests.test_releasectl
	$(MAKE) release-tag-check TOOL=$(TOOL) RELEASE_TAG="$(RELEASE_TAG)"
	$(MAKE) $(TOOL)-test GO_TEST_FLAGS=-short

.PHONY: release-local
release-local:
	$(call require_release_tool,release-local)
	$(call require_release_tag,release-local)
	@workflow="$$( $(RELEASECTL) workflow-from-tag --tag "$(RELEASE_TAG)" )"; \
	$(MAKE) release-preflight TOOL=$(TOOL) RELEASE_TAG="$(RELEASE_TAG)"; \
	matrix_json="$$( $(RELEASECTL) host-matrix --tool $(TOOL) --workflow "$$workflow" --goos "$(HOST_GOOS)" )"; \
	commands="$$( MATRIX_JSON="$$matrix_json" $(PYTHON) -c 'import json, os; data = json.loads(os.environ["MATRIX_JSON"]); [print(entry["build_command"]) or print(entry["package_command"]) for entry in data["include"]]' )"; \
	while IFS= read -r command; do \
		[ -n "$$command" ] || continue; \
		echo "-> $$command"; \
		eval "$$command"; \
	done <<< "$$commands"
