# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

SUBPROJECTS := common cluster-agent hardware-discovery-agent node-agent platform-observability-agent platform-telemetry-agent platform-update-agent reporting-agent
NO_BUILD := common platform-observability-agent
NO_FUZZ := common platform-observability-agent
NO_LINT := platform-observability-agent
NO_TAR := common platform-update-agent
NO_UNIT := platform-observability-agent
NO_PACKAGE := common
NO_CLEAN := common

.PHONY: build-agents test-agents fuzztest-agents clean-agents help lint-agents package-agents tarball-agents

build-agents:
	@# Help: runs `build` target for each sub directory/agents
	@for s in $(filter-out $(NO_BUILD), $(SUBPROJECTS)); do \
		echo "Building $$s"; \
		$(MAKE) -C $$s build; \
	done

test-agents:
	@# Help: runs `test` target for each sub directory/agents
	@for s in $(filter-out $(NO_UNIT), $(SUBPROJECTS)); do \
		echo "Testing $$s"; \
	    $(MAKE) -C $$s test; \
	done

fuzztest-agents:
	@# Help: runs `fuzztest` target for each sub directory/agents
	@for s in $(filter-out $(NO_FUZZ), $(SUBPROJECTS)); do \
		echo "Fuzzing $$s"; \
	    $(MAKE) -C $$s fuzztest; \
	done

lint-agents:
	@# Help: runs `lint` target for each sub directory/agents
	@for s in $(filter-out $(NO_LINT), $(SUBPROJECTS)); do \
		echo "Linting $$s"; \
	    $(MAKE) -C $$s lint; \
	done

tarball-agents:
	@# Help: runs `tarball` target for each sub directory/agents
	@for s in $(filter-out $(NO_TAR), $(SUBPROJECTS)); do \
		echo "Tarballing $$s"; \
	    $(MAKE) -C $$s tarball; \
	done

package-agents:
	@# Help: runs `package` target for each sub directory/agents
	@for s in $(filter-out $(NO_PACKAGE), $(SUBPROJECTS)); do \
		echo "Packaging $$s"; \
	    $(MAKE) -C $$s package; \
	done

clean-agents:
	@# Help: runs `clean` target for each sub directory/agents
	@for s in $(filter-out $(NO_CLEAN), $(SUBPROJECTS)); do \
		echo "Cleaning $$s"; \
	    $(MAKE) -C $$s clean; \
	done

help:
	@printf "%-20s %s\n" "Target" "Description"
	@printf "%-20s %s\n" "------" "-----------"
	@make -pqR : 2>/dev/null \
        | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' \
        | sort \
        | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' \
        | xargs -I _ sh -c 'printf "%-20s " _; make _ -nB | (grep -i "^# Help:" || echo "") | tail -1 | sed "s/^# Help: //g"'
