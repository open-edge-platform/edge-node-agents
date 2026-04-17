// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package statusService

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"

	pb "github.com/open-edge-platform/edge-node-agents/common/pkg/api/status/proto"
	pv "github.com/open-edge-platform/edge-node-agents/common/pkg/protovalidator"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/logger"
	"google.golang.org/grpc"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
)

// Initialize logger
var log = logger.Logger

type StatusValue struct {
	Status    pb.Status
	Timestamp int64
}

type StatusService struct {
	pb.UnimplementedStatusServiceServer
	// map of agent name to status,timestamp (when last status was received)
	statusMap sync.Map
	// map of endpoint name to status,timestamp (when last checked) - READY implies up, NOT_READY implies down
	nwStatusMap sync.Map
	// set of registered agents, map is used as a set for fast lookups
	agents map[string]struct{}
	// interval for status checks
	statusInterval time.Duration
}

type CmdExecutor = func(name string, args ...string) *exec.Cmd

func (s *StatusService) ReportStatus(ctx context.Context, in *pb.ReportStatusRequest) (*pb.ReportStatusResponse, error) {

	if err := pv.ValidateMessage(in); err != nil {
		log.Errorf("error validating ReportStatusRequest : %v", err)
		return nil, err
	}

	if _, exists := s.agents[in.AgentName]; !exists {
		return nil, fmt.Errorf("agent %s not known", in.AgentName)
	}

	var status StatusValue
	sValue, exists := s.statusMap.Load(in.AgentName)

	// If there isn't a status value for this agent, create one
	if !exists {
		status = StatusValue{}
	} else {
		status = sValue.(StatusValue)
	}

	// Overwrite the received status & timestamp blindly. Logic for missed
	// status and tolerance should be implemented at the point of aggregation
	status.Status = in.Status
	status.Timestamp = time.Now().Unix()
	s.statusMap.Store(in.AgentName, status)

	return &pb.ReportStatusResponse{}, nil
}

func (s *StatusService) GetStatusInterval(ctx context.Context, in *pb.GetStatusIntervalRequest) (*pb.GetStatusIntervalResponse, error) {
	if err := pv.ValidateMessage(in); err != nil {
		log.Errorf("error validating GetIntervalStatusRequest : %v", err)
		return nil, err
	}
	return &pb.GetStatusIntervalResponse{IntervalSeconds: int32(s.statusInterval.Seconds())}, nil
}

func InitStatusService(confs *config.NodeAgentConfig) (*grpc.Server, *StatusService) {

	grpcServer := grpc.NewServer()
	statusService := StatusService{
		agents:         make(map[string]struct{}),
		statusInterval: confs.Onboarding.HeartbeatInterval,
	}

	for _, agent := range confs.Status.ServiceClients {
		statusService.agents[agent] = struct{}{}
		statusService.statusMap.Store(agent, StatusValue{
			Status:    pb.Status_STATUS_UNSPECIFIED,
			Timestamp: time.Now().Unix(),
		})
	}

	for _, endpoint := range confs.Status.NetworkEndpoints {
		statusService.nwStatusMap.Store(endpoint.Name, StatusValue{
			Status:    pb.Status_STATUS_UNSPECIFIED,
			Timestamp: time.Now().Unix(),
		})
	}

	pb.RegisterStatusServiceServer(grpcServer, &statusService)

	log.Infoln("Status service initialized")

	return grpcServer, &statusService
}

func (s *StatusService) GatherStatus(confs *config.NodeAgentConfig) (string, bool) {

	// Initialized at 1 to account for node agent itself
	counter := 1
	total := 1
	currentTime := time.Now().Unix()

	// Count services' status here & get contribution to net counts
	active, totalServices, unhealthy := CheckServicesStatus(confs.Status.OutboundClients, exec.Command)

	counter += active
	total += totalServices

	nwInterval := int64(confs.Status.NetworkStatusInterval.Seconds()) // Interval for network polling
	hbInterval := int64(confs.Onboarding.HeartbeatInterval.Seconds()) // Interval for heartbeats

	// Collect network status
	s.nwStatusMap.Range(func(key, value interface{}) bool {
		total++
		statusValue := value.(StatusValue)
		// Tolerate 20 seconds(equivalent to 2 default heartbeat cycles) of delay with network polling
		if currentTime-statusValue.Timestamp <= nwInterval+(2*hbInterval) && statusValue.Status == pb.Status_STATUS_READY {
			counter++
		} else { // collect names of agents that are not running
			unhealthy = append(unhealthy, key.(string))
		}
		return true
	})

	s.statusMap.Range(func(key, value interface{}) bool {
		total++
		statusValue := value.(StatusValue)
		// Tolerate 2 missed status messages
		if currentTime-statusValue.Timestamp <= (2*hbInterval) && statusValue.Status == pb.Status_STATUS_READY {
			counter++
		} else { // collect names of agents that are not running
			unhealthy = append(unhealthy, key.(string))
		}
		return true
	})

	log.Infof("%d of %d components running", counter, total)
	if counter != total {
		log.Warnf("Unhealthy components : %v", unhealthy)
	}

	// Return formatted string to HRM, boolean value for instance status
	return fmt.Sprintf("%d of %d components running", counter, total), counter == total
}

// CheckServicesStatus checks the status of the services, custom CmdExecutor
// is used to enhance testability
func CheckServicesStatus(services []string, command CmdExecutor) (int, int, []string) {
	active := 0
	total := len(services)
	unhealthy := []string{}

	for _, service := range services {
		cmd := command("systemctl", "is-active", service)
		output, err := cmd.Output()
		if err != nil {
			log.Errorf("Failed to check status of service %s: %v", service, err)
			unhealthy = append(unhealthy, service)
			continue
		}

		if string(output) == "active\n" {
			active++
		} else {
			unhealthy = append(unhealthy, service)
		}
	}

	return active, total, unhealthy
}

// PollNetworkEndpoints polls the given endpoints and logs their status
func (s *StatusService) PollNetworkEndpoints(ctx context.Context, endpoints []config.NetworkEndpoint) {
	for _, endpoint := range endpoints {
		select {
		case <-ctx.Done():
			return
		default:
			var status StatusValue
			sValue, exists := s.nwStatusMap.Load(endpoint.Name)

			// If there isn't a status value for this endpoint, create one
			if !exists {
				status = StatusValue{}
			} else {
				status = sValue.(StatusValue)
			}
			status.Status = pb.Status_STATUS_NOT_READY

			err := ProbeEndpoint(ctx, endpoint.URL)
			if err != nil {
				log.Errorf("Failed to poll endpoint %s: %v", endpoint, err)
				status.Timestamp = time.Now().Unix()
				s.nwStatusMap.Store(endpoint.Name, status)
				continue
			}
			log.Debugf("Endpoint %s is reachable", endpoint.Name)
			status.Status = pb.Status_STATUS_READY
			status.Timestamp = time.Now().Unix()
			s.nwStatusMap.Store(endpoint.Name, status)
		}
	}
}

func ProbeEndpoint(ctx context.Context, endpoint string) error {

	if strings.HasPrefix(endpoint, "oci://") {
		ref, err := remote.NewRepository(strings.TrimPrefix(endpoint, "oci://"))
		if err != nil {
			log.Errorf("Failed to create repository for endpoint %s: %v", endpoint, err)
			return err
		}
		parts := strings.Split(endpoint, ":")
		tag := "main"
		if len(parts) > 1 {
			tag = parts[len(parts)-1]
		}
		_, _, err = oras.Fetch(ctx, ref, tag, oras.FetchOptions{})
		if err != nil {
			return err
		}
		log.Debugf("OCI endpoint %s is reachable", endpoint)
		return nil
	} else if strings.HasPrefix(endpoint, "https://") || strings.HasPrefix(endpoint, "http://") {
		u, err := url.Parse(endpoint)
		if err != nil {
			log.Errorf("Failed to parse URL for endpoint %s", endpoint)
			return err
		}
		resp, err := http.Get(u.String())
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			log.Debugf("Endpoint %s is reachable", endpoint)
			return nil
		} else {
			log.Warnf("Endpoint %s returned status %d", endpoint, resp.StatusCode)
			return fmt.Errorf("endpoint %s returned status %d", endpoint, resp.StatusCode)
		}
	}

	log.Warnf("Unsupported endpoint format: %s", endpoint)
	return fmt.Errorf("unsupported endpoint format: %s", endpoint)
}
