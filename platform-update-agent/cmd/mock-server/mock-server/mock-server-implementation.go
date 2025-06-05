// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package mock_server

import (
	"context"
	"fmt"
	"log"
	"math"
	"net"
	"strings"
	"sync"
	"time"

	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

var globalIteration = 0

type ServerType int

const (
	UBUNTU ServerType = iota
	EMT
	DEBIAN
)

// String is part of flag.Value interface
func (s *ServerType) String() string {
	return [...]string{"UBUNTU", "EMT", "DEBIAN"}[*s]
}

// Set is part of flag.Value interface
func (s *ServerType) Set(value string) error {
	switch strings.ToUpper(value) {
	case "UBUNTU":
		*s = UBUNTU
	case "EMT":
		*s = EMT
	case "DEBIAN":
		*s = DEBIAN
	default:
		return fmt.Errorf("invalid ServerType: %s", value)
	}
	return nil
}

type server struct {
	pb.UnimplementedMaintmgrServiceServer
	listen     string
	serverType ServerType
}

// NewGrpcServer creates a new gRPC server for mock MM
// it will listen on the provided address and use the provided certificates
// serverType is used to determine the type of updates the server will serve
func NewGrpcServer(grpcAddr string, certsPath string, serverType ServerType) (*grpc.Server, net.Listener) {
	s := &server{
		listen:     grpcAddr,
		serverType: serverType,
	}

	lis, err := net.Listen("tcp", s.listen)
	if err != nil {
		log.Fatal("Error creating server listener - " + err.Error())
	}

	tlsCreds, err := credentials.NewServerTLSFromFile(certsPath+"/server-cert.pem", certsPath+"/server-key.pem")
	if err != nil {
		log.Fatal("Not able to create credentials for server - " + err.Error())
	}
	grpcServer := grpc.NewServer(grpc.Creds(tlsCreds))
	pb.RegisterMaintmgrServiceServer(grpcServer, s)

	return grpcServer, lis
}

// StartMockServer starts a mock MM server; the type of updates it will serve is determined by the serverType parameter
func StartMockServer(serverType ServerType) {
	wg := &sync.WaitGroup{}

	log.Println("Starting up...")
	grpcAddr := "localhost:8089"

	mms, mml := NewGrpcServer(grpcAddr, "../../mocks", serverType)

	if mms == nil {
		log.Println("Failed to create maintenance manager gRPC server.")
	}

	log.Println("Starting gRPC server on " + grpcAddr)

	wg.Add(1)
	go func() {
		if err := RunGrpcServer(mms, mml); err != nil {
			log.Println("Failed to run maintenance gRPC server")
		}
	}()

	wg.Wait()
}

func RunGrpcServer(server *grpc.Server, lis net.Listener) error {
	if err := server.Serve(lis); err != nil {
		log.Println("Error running nb server")
		return err
	}

	return nil
}

func generateDailyRepeatedSchedule(startTime time.Time, duration time.Duration) *pb.RepeatedSchedule {
	return &pb.RepeatedSchedule{
		DurationSeconds: uint32(math.Round(duration.Seconds())),
		CronMinutes:     fmt.Sprintf("%d", startTime.UTC().Minute()),
		CronHours:       fmt.Sprintf("%d", startTime.UTC().Hour()),
		CronDayMonth:    "*",
		CronMonth:       "*",
		CronDayWeek:     "*",
	}
}

var startSeconds uint64

func (s *server) PlatformUpdateStatus(_ context.Context, req *pb.PlatformUpdateStatusRequest) (*pb.PlatformUpdateStatusResponse, error) {
	var singleSchedule *pb.SingleSchedule
	if globalIteration%3 == 0 {
		singleSchedule = &pb.SingleSchedule{
			StartSeconds: uint64(time.Now().Add(time.Second * 5).Unix()),
			EndSeconds:   uint64(time.Now().Add(time.Second * 9).Unix()),
		}
	} else {
		singleSchedule = nil
	}
	globalIteration++

	log.Printf("Received PlatformUpdateStatus request for server type %v: %v\n", s.serverType, req)

	var response *pb.PlatformUpdateStatusResponse
	switch s.serverType {
	case UBUNTU, DEBIAN:
		response = &pb.PlatformUpdateStatusResponse{
			OsType: pb.PlatformUpdateStatusResponse_OS_TYPE_MUTABLE,
			UpdateSource: &pb.UpdateSource{
				KernelCommand: "abc def",
				OsRepoUrl:     "http://linux-ftp.fi.intel.com/pub/mirrors/ubuntu",
				CustomRepos: []string{
					"Types: deb\nURIs: https://files.internal.example.com\nSuites: example\nComponents: release\nSigned-By:\n -----BEGIN PGP PUBLIC KEY BLOCK-----\n .\n mQINBGSr79IBEADkblVsUPjZKyr9xGT7Hv4/94NDOOJIr8PZSKHnOpDNcev4lmSRExMNzxLsckOT\n MqY+nyRtQ8fBxwIZLpRxXJBzsU/1D/JTqFTgnTV0CN73kkZbN6/0QKQOdUo9i3y4zxjZhqC6PRoX\n ijMP+lcvkI1ixj0rwgMY6w5qpeCDZD1W1YcHQXyfP9S21sd/wJ6niAGYv8Eyxz+uQniPLqSSjfp1\n u9aJw1YO8kaTLx7gBJJU810kyQxi2q+gVvMDmITj2ufWm+Tbnj5j8+YnEmdmyQwPp27koHmeeeC7\n EKxahvATcppd15h8ZMNL8Clbdhg71yTMWjYcVQ01RvceA6/5zmX8/yC5m6HdtgdhYgcgPQ5rqYca\n E8MjoPcVA04y5+6z1P7/hXyVQzIhSDN1RdiqD1mxDp3pKXXn4Wv18B3oyyo74HAVXefo7iP1cYVA\n UUKMH8JsWCpX3DTwYzrmKWy1s6gzyv56YfvGZ36Hx+YuW7tV1rrQ35kSnhKRmD0msaHeG8DgB3in\n tV/1TzSLT8p+b2QPOD/EJNXfW+wNMiOPZ/imyagfIBjJg4inakmZJrqSS5Va2dK/KCIniNTunuc5\n iQeZSEbApwKKJQ+0L2FjSkjYHq8d/z3DVSv1JU8nCAf30YpmAjijyn62EJeuarH7P4I9RHy9Sm9e\n g1t0fcuzJa/UWwARAQABtQAsQ049bGVkZ2VwYXJrLWRlYmlhbi1zaWduaW5nLWtleS1ncGctbm9u\n LXByb2SJAj4EEwEIACgFAmSr79ICGwMFCQHhM4AGCwkIBwMCBhUIAgkKCwQWAgMBAh4BAheAAAoJ\n EAovmJ4jairgSHwQAKwdR/f5Agoko0sdqi8SI7OvWDPIfa0ZWrTBeKxICUk7gzt1I1n0Mj5ZXpGd\n NJHbpBsdsw486NiQLCXYX/JkiYN3wMb04L7uFjz3rw3PsaAV/N4/sybfB6GYLy8w1j2xdHRW8dkk\n y35sF4pnqAUDW/eJJpYsM4PiMsXCS9i3M2NLtKMkkDZXyZJDEGpJ1/M2K6NU/eJi74HIQWmMrsB2\n RIu3J0WP1bXPPInoc28ccn98T9cA1n9egGDSpuyq1N6M+oyxfdvSh1rk3K1oSZvsUXcjbqJQPAZ0\n MjPVp4nH0YnUfj66HPnrqDaloqPYJ3in1FNwc8E2LCNIvEws82fO3HujUOKjiwZeax+ve1nOtRZx\n BxBkfFLWdrYTIhIS8TtShJw5s/AC15P2purZDSQyqRlS1RlFjb1oUCvwGjFgw9nIDLTyjbxFxhHQ\n L5idFWlesJl4SmYo3+7ry6OodxVshxd+tsgS1ktHGhpJk4+2uzitEaKimnzVq5R+ZTiOUas86KDQ\n V74ZAxrzL4hwLnbMk5qnRenH+KT2XOfCVdVDKEVvuLkGR3C7k7gT9IhV3jWZ/CKjRWM+SOwMLUhu\n vCCuuhGENRKlv4UJ/eZyQYnSh3vLbG2FrRJ6PtOGlpMw9kFTer0ujA9rbWBxEySAVeRcVd9zwHEL\n K/GMsEjjy4KQyWKV\n =/Xiv\n -----END PGP PUBLIC KEY BLOCK-----",
				},
			},
			UpdateSchedule: &pb.UpdateSchedule{
				SingleSchedule: singleSchedule,
				RepeatedSchedules: []*pb.RepeatedSchedule{
					{
						DurationSeconds: 300,
						CronMinutes:     "*/1",
						CronHours:       "*",
						CronDayMonth:    "*",
						CronMonth:       "*",
						CronDayWeek:     "*",
					},
				},
			},
			InstalledPackages: "intel-opencl-icd\nnet-tools",
		}
	case EMT:
		// setup for emt to check download + update timing; assuming short download / immediate download windows
		var singleSchedule *pb.SingleSchedule
		var repeatedSchedules []*pb.RepeatedSchedule

		if startSeconds == 0 {
			startSeconds = (uint64)(time.Now().Add(30 * time.Second).Unix())
		}
		singleSchedule = &pb.SingleSchedule{
			StartSeconds: startSeconds,
			EndSeconds:   startSeconds + 30,
		}
		repeatedSchedules = []*pb.RepeatedSchedule{
			generateDailyRepeatedSchedule(time.Now().Add(1*time.Hour), 5*time.Minute),
			generateDailyRepeatedSchedule(time.Now().Add(2*time.Hour), 5*time.Minute),
		}

		response = &pb.PlatformUpdateStatusResponse{
			OsType: pb.PlatformUpdateStatusResponse_OS_TYPE_IMMUTABLE,
			UpdateSchedule: &pb.UpdateSchedule{
				SingleSchedule:    singleSchedule,
				RepeatedSchedules: repeatedSchedules,
			},
			OsProfileUpdateSource: &pb.OSProfileUpdateSource{
				OsImageUrl:     "https://www.example.com/",
				OsImageId:      "example-image-id",
				OsImageSha:     "example-image-sha",
				ProfileName:    "example-profile-name",
				ProfileVersion: "example-profile-version",
			},
		}
	default:
		return nil, status.Error(codes.Internal, fmt.Sprintf("Server has unknown type configured: %v", s.serverType))
	}

	log.Printf("Sending PlatformUpdateStatus response for server type %v: %v\n", s.serverType, response)

	return response, nil
}
