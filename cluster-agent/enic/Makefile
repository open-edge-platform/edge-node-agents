# Define default values for CPU, memory, and host
CPUS ?= 4
MEMORY ?= 8G
CLUSTER_ORCH_HOST ?= cluster-orch-node.kind.internal
CLUSTER_ORCH_HOST_IP ?= 172.18.255.237
ARGOCD_IP ?= 172.18.255.236
ARGOCD_PASSWORD ?= "ChangeMeOn1stLogin!"
TARGETARCH ?= amd64

# Generate a unique value based on the current time
RANDOM_VALUE := $(shell date +%s%N)

DOCKER_BUILD_ARGS ?= \
	--build-arg http_proxy="$(http_proxy)" --build-arg https_proxy="$(https_proxy)" \
	--build-arg no_proxy="$(no_proxy)" --build-arg HTTP_PROXY="$(http_proxy)" \
	--build-arg HTTPS_PROXY="$(https_proxy)" --build-arg NO_PROXY="$(no_proxy)" \
	--build-arg TARGETARCH=${TARGETARCH}

.PHONY: vendor
vendor:  ## Build vendor directory of module dependencies.
	cd ../ && GOPRIVATE=github.com/open-edge-platform/* go mod vendor

# Target to build the Docker image
build-enic-docker: vendor ## Build the ENiC Docker image
	# Set the target architecture and build the Docker image
	docker build ${DOCKER_BUILD_ARGS} -t registry.espdprod.infra-host.com/edge-orch/cluster/cluster-agent:lw-enic -f Dockerfile ../

# Target to clean up the environment
clean-enic-docker: ## Clean up the ENiC Docker container
	# Stop the running container named cluster-agent if it exists
	docker stop cluster-agent || true;
	# Remove the container named cluster-agent if it exists
	docker rm cluster-agent || true
	# Remove any directories starting with rancher- using sudo
	sudo rm -rf rancher-*

clean-enic-pod: ## Clean ENiC pod up the k8s environment
	kubectl delete -f enic-ss.yaml --wait || true
	kubectl wait --for=delete pod -l app=cluster-agent --timeout=300s || true

load-enic-docker: ## Load the ENiC docker image into the kind cluster
	kind load docker-image registry.espdprod.infra-host.com/edge-orch/cluster/cluster-agent:lw-enic

# Run light weight ENiC as a standalone docker container
run-enic-docker: clean-enic-docker ## Run ENiC as a standalone docker container
	# Run a new Docker container with the specified options
	docker run -td --privileged --cpus="$(CPUS)" -m="$(MEMORY)"  -h cluster-agent \
		--add-host=$(CLUSTER_ORCH_HOST):$(CLUSTER_ORCH_HOST_IP) \
		-v ./rancher-$(RANDOM_VALUE):/var/lib/rancher:rw  \
		--name=cluster-agent registry.espdprod.infra-host.com/edge-orch/cluster/cluster-agent:lw-enic /lib/systemd/systemd

run-enic-pod: clean-enic-pod ## Run ENiC as a pod in k8s
	# This script adds 'ALLOW_MISSING_CLIENTS' env to southbound-handler with cluster-agent client
	# info so that southbound-handler can skip rbac for cluster-agent
	./add_env_var.sh
	kubectl apply -f enic-ss.yaml
	kubectl rollout status statefulset/cluster-agent

log-enic: ## Log cluster-agent logs from ENiC pod in k8s
	kubectl exec  cluster-agent-0 -- journalctl -feu cluster-agent

exec-enic: ## Exec into enic pod
	kubectl exec -it cluster-agent-0 -- bash

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
