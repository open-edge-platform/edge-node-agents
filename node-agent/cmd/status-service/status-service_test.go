// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package statusService

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"sync"
	"testing"
	"time"

	pb "github.com/open-edge-platform/edge-node-agents/common/pkg/api/status/proto"
	"github.com/open-edge-platform/edge-node-agents/node-agent/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestReportStatus(t *testing.T) {
	agents := []string{"test-agent-one", "test-agent-two", "test-agent-name-with-more-than-forty-chars"}
	statusService := StatusService{
		agents: make(map[string]struct{}),
	}

	for _, agent := range agents {
		statusService.agents[agent] = struct{}{}
	}

	tests := []struct {
		name      string
		agentName string
		status    pb.Status
		wantErr   bool
	}{
		{
			name:      "Known agent",
			agentName: "test-agent-one",
			status:    pb.Status_STATUS_READY,
			wantErr:   false,
		},
		{
			name:      "Unknown agent",
			agentName: "test-agent-three",
			status:    pb.Status_STATUS_READY,
			wantErr:   true,
		},
		{
			name:      "Validation failure",
			agentName: "test-agent-name-with-more-than-forty-chars", //long agent name
			status:    pb.Status_STATUS_READY,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &pb.ReportStatusRequest{
				AgentName: tt.agentName,
				Status:    tt.status,
			}
			resp, err := statusService.ReportStatus(context.Background(), req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				sValue, exists := statusService.statusMap.Load(tt.agentName)
				assert.True(t, exists)
				statusValue := sValue.(StatusValue)
				assert.Equal(t, tt.status, statusValue.Status)
				assert.WithinDuration(t, time.Now(), time.Unix(statusValue.Timestamp, 0), time.Second)
			}
		})
	}
}

func TestInitStatusService(t *testing.T) {
	agents := []string{"agent-one", "agent-two"}
	endpoints := []config.NetworkEndpoint{
		{Name: "endpoint1", URL: "http://endpoint1.com"},
	}
	cfg := config.NodeAgentConfig{
		Status: config.ConfigStatus{
			ServiceClients:   agents,
			NetworkEndpoints: endpoints,
		},
		Onboarding: config.ConfigOnboarding{
			HeartbeatInterval: 1 * time.Second,
		},
	}

	grpcServer, statusService := InitStatusService(&cfg)

	assert.NotNil(t, grpcServer)
	assert.NotNil(t, statusService)
	assert.Equal(t, len(agents), len(statusService.agents))
	count := 0
	statusService.statusMap.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, len(agents), count)
	assert.Equal(t, statusService.statusInterval, 1*time.Second)

	for _, agent := range agents {
		_, exists := statusService.agents[agent]
		assert.True(t, exists)
		val, exists := statusService.statusMap.Load(agent)
		assert.True(t, exists)
		assert.Equal(t, pb.Status_STATUS_UNSPECIFIED, val.(StatusValue).Status)
	}

	var nwEndpoints []string
	for _, endpoint := range endpoints {
		nwEndpoints = append(nwEndpoints, endpoint.Name)
	}

	for _, endpoint := range nwEndpoints {
		val, exists := statusService.nwStatusMap.Load(endpoint)
		assert.True(t, exists)
		assert.Equal(t, pb.Status_STATUS_UNSPECIFIED, val.(StatusValue).Status)
	}
}

func TestGatherStatus(t *testing.T) {

	cfg := &config.NodeAgentConfig{
		Status: config.ConfigStatus{
			OutboundClients:       []string{},
			NetworkStatusInterval: 60 * time.Second,
		},
		Onboarding: config.ConfigOnboarding{
			HeartbeatInterval: 10 * time.Second,
		},
	}

	tests := []struct {
		name        string
		statusMap   map[string]StatusValue
		nwStatusMap map[string]StatusValue
		statusMsg   string
		goodStatus  bool
	}{
		{
			name: "All agents ready",
			statusMap: map[string]StatusValue{
				"agent-one": {Status: pb.Status_STATUS_READY, Timestamp: time.Now().Unix()},
				"agent-two": {Status: pb.Status_STATUS_READY, Timestamp: time.Now().Unix()},
			},
			nwStatusMap: map[string]StatusValue{
				"nwEp1": {Status: pb.Status_STATUS_READY, Timestamp: time.Now().Unix()},
			},
			statusMsg:  "4 of 4 components running",
			goodStatus: true,
		},
		{
			name: "Some agents not ready",
			statusMap: map[string]StatusValue{
				"agent-one": {Status: pb.Status_STATUS_READY, Timestamp: time.Now().Unix()},
				"agent-two": {Status: pb.Status_STATUS_NOT_READY, Timestamp: time.Now().Unix()},
			},
			nwStatusMap: map[string]StatusValue{
				"nwEp1": {Status: pb.Status_STATUS_READY, Timestamp: time.Now().Unix()},
			},
			statusMsg:  "3 of 4 components running",
			goodStatus: false,
		},
		{
			name: "Nw endpoint not ready",
			statusMap: map[string]StatusValue{
				"agent-one": {Status: pb.Status_STATUS_READY, Timestamp: time.Now().Unix()},
				"agent-two": {Status: pb.Status_STATUS_NOT_READY, Timestamp: time.Now().Unix()},
			},
			nwStatusMap: map[string]StatusValue{
				"nwEp1": {Status: pb.Status_STATUS_UNSPECIFIED, Timestamp: time.Now().Unix()},
			},
			statusMsg:  "2 of 4 components running",
			goodStatus: false,
		},
		{
			name: "Nw endpoint status outdated",
			statusMap: map[string]StatusValue{
				"agent-one": {Status: pb.Status_STATUS_READY, Timestamp: time.Now().Unix() - 5},
				"agent-two": {Status: pb.Status_STATUS_READY, Timestamp: time.Now().Unix() - 15},
			},
			nwStatusMap: map[string]StatusValue{
				"nwEp1": {Status: pb.Status_STATUS_READY, Timestamp: time.Now().Unix() - 59},
				"nwEp2": {Status: pb.Status_STATUS_UNSPECIFIED, Timestamp: time.Now().Unix() - 109},
			},
			statusMsg:  "4 of 5 components running",
			goodStatus: false,
		},
		{
			name: "Some agents status outdated",
			statusMap: map[string]StatusValue{
				"agent-one": {Status: pb.Status_STATUS_READY, Timestamp: time.Now().Unix() - 25},
				"agent-two": {Status: pb.Status_STATUS_READY, Timestamp: time.Now().Unix() - 15},
			},
			statusMsg:  "2 of 3 components running",
			goodStatus: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statusService := StatusService{
				agents: make(map[string]struct{}),
			}

			for agent, status := range tt.statusMap {
				statusService.statusMap.Store(agent, status)
				statusService.agents[agent] = struct{}{}
			}
			for nwEp, status := range tt.nwStatusMap {
				statusService.nwStatusMap.Store(nwEp, status)
			}

			gotString, gotBool := statusService.GatherStatus(cfg)
			assert.Equal(t, tt.statusMsg, gotString)
			assert.Equal(t, tt.goodStatus, gotBool)
		})
	}
}

func MockCommandExecutor(name string, arg ...string) ([]byte, error) {
	if name == "systemctl" && len(arg) == 2 && arg[0] == "is-active" {
		switch arg[1] {
		case "activeService":
			return []byte("active"), nil
		case "inactiveService":
			return []byte("inactive"), nil
		default:
			return nil, errors.New("unknown service")
		}
	}
	return nil, errors.New("invalid command")
}

func TestCheckServicesStatus(t *testing.T) {
	tests := []struct {
		name       string
		services   []string
		wantActive int
		wantTotal  int
	}{
		{
			name:       "All services active",
			services:   []string{"activeService", "activeService"},
			wantActive: 2,
			wantTotal:  2,
		},
		{
			name:       "Some services inactive",
			services:   []string{"activeService", "inactiveService"},
			wantActive: 1,
			wantTotal:  2,
		},
		{
			name:       "All services inactive",
			services:   []string{"inactiveService", "inactiveService"},
			wantActive: 0,
			wantTotal:  2,
		},
		{
			name:       "Unknown service",
			services:   []string{"unknownService"},
			wantActive: 0,
			wantTotal:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			active, total, unhealthy := CheckServicesStatus(tt.services, func(name string, args ...string) *exec.Cmd {
				output, _ := MockCommandExecutor(name, args...)
				return exec.Command("echo", string(output)) // echo appends a terminating newline character
			})
			assert.Equal(t, tt.wantActive, active)
			assert.Equal(t, tt.wantTotal, total)
			assert.Equal(t, tt.wantTotal-tt.wantActive, len(unhealthy))
		})
	}
}
func TestGetStatusInterval(t *testing.T) {
	statusService := StatusService{
		statusInterval: 30 * time.Second,
	}

	req := &pb.GetStatusIntervalRequest{AgentName: "test-agent"}
	resp, err := statusService.GetStatusInterval(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(30), resp.IntervalSeconds)
}

func TestGetStatusInterval_prtoVal(t *testing.T) {
	statusService := StatusService{
		statusInterval: 30 * time.Second,
	}

	// numers not allowed in name
	req := &pb.GetStatusIntervalRequest{AgentName: "agent1"}
	resp, err := statusService.GetStatusInterval(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestPollNetworkEndpointsHTTP(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantStatus pb.Status
	}{
		{
			name:       "Endpoint returns 200",
			statusCode: 200,
			wantStatus: pb.Status_STATUS_READY,
		},
		{
			name:       "Endpoint returns 500",
			statusCode: 500,
			wantStatus: pb.Status_STATUS_NOT_READY,
		},
		{
			name:       "Endpoint returns 404",
			statusCode: 404,
			wantStatus: pb.Status_STATUS_NOT_READY,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock server
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer mockServer.Close()

			statusService := StatusService{
				nwStatusMap: sync.Map{},
			}
			statusService.nwStatusMap.Store("mockEndpoint", StatusValue{Status: pb.Status_STATUS_UNSPECIFIED})

			nwConfigs := []config.NetworkEndpoint{
				{Name: "mockEndpoint", URL: mockServer.URL},
			}

			statusService.PollNetworkEndpoints(context.Background(), nwConfigs)

			val, exists := statusService.nwStatusMap.Load("mockEndpoint")
			assert.True(t, exists)
			assert.Equal(t, tt.wantStatus, val.(StatusValue).Status)
		})
	}
}

func TestPollNetworkEndpoints_incorrectURL(t *testing.T) {

	statusService := StatusService{
		nwStatusMap: sync.Map{},
	}
	statusService.nwStatusMap.Store("mockEndpoint", StatusValue{Status: pb.Status_STATUS_UNSPECIFIED})

	nwConfigs := []config.NetworkEndpoint{
		{Name: "mockEndpoint", URL: "badurl"},
	}

	statusService.PollNetworkEndpoints(context.Background(), nwConfigs)

	val, exists := statusService.nwStatusMap.Load("mockEndpoint")
	assert.True(t, exists)
	assert.Equal(t, pb.Status_STATUS_NOT_READY, val.(StatusValue).Status)

}
