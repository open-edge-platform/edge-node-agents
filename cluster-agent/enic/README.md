# Light-weight Edge Node in Container (ENiC)

Light weight ENiC with just a tweaked cluster-agent running as systemd service suitable for Cluster Orch integration tests

# Usage
Run `make help` details

## Some useful commands
1. Run `make build-docker` to build cluster-agent
2. Run `make load-docker` everytime ENiC docker image is updated to load the updated docker image to K8s cluster
3. Run `make run-pod` to deploy the ENiC pod into K8s cluster and also apply disable JWT auth at Southbound handler.
The light weight ENiC does not use JWT auth, so we need to disable the same at SB Handler
4. Run `make log-k8s` to view cluste-agent logs at ENiC

# Configuration
The cluster agent configuration is defined in cluster-agent.yaml file. Some useful configurations for the test
- GUID: GUID of the EN.
- clusterOrchestratorURL: Endpoint of the SB gRPC handler. In this case since ENiC runs in same k8s cluster as
SB Handler, it points to SB Handler's DNS name
