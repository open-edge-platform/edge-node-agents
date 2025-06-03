// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	mock_server "github.com/open-edge-platform/edge-node-agents/platform-update-agent/cmd/mock-server/mock-server"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/comms"
	puaConfig "github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/downloader"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/metadata"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/scheduler"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/updater"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/utils"
	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MockDownloadExecutor struct{}

func (m *MockDownloadExecutor) Download(
	ctx context.Context,
	prependToImageURL string,
	source *pb.OSProfileUpdateSource,
) error {
	return nil
}

func createNullLogger() *logrus.Entry {
	logger := logrus.New()
	logger.Out = io.Discard
	return logger.WithFields(logrus.Fields{"test": true})
}

func Fuzz_PlatformUpdateStatus_FuzzGrpcCommunication(f *testing.F) {
	client := prepareClientAndServer(f)
	f.Add("ABCDEFGHIKLMNOPQRSTVXYZ+-/*", "profileName", "1.0.0", "osImage123")
	f.Add("abcdefghiklmnopqrstvxyz+-/*", "testProfile", "2.1.0", "osImage456")
	f.Add(uuid.New().String(), "fuzzProfile", "0.0.1", uuid.New().String())

	f.Fuzz(func(t *testing.T, statusDetail, profileName, profileVersion, osImageId string) {
		status := &pb.UpdateStatus{
			StatusType:     pb.UpdateStatus_StatusType(rand.Int() % 5),
			StatusDetail:   statusDetail,
			ProfileName:    profileName,
			ProfileVersion: profileVersion,
			OsImageId:      osImageId,
		}

		hostGUID := uuid.New().String()

		_, err := client.PlatformUpdateStatus(context.TODO(), status, hostGUID)

		if err != nil && !strings.Contains(err.Error(), "string field contains invalid UTF-8") {
			t.Fatalf("fuzzing error for status %+v, hostGUID %v - %v", status, hostGUID, err)
			t.Fail()
		}
	})
}

func prepareClientAndServer(f *testing.F) *comms.Client {
	const serviceAddr = "127.0.0.1"
	// Use a port in the dynamic/private range (49152–65535)
	servicePort := fmt.Sprintf("%d", 49152+rand.Intn(65535-49152+1))
	serviceURL := serviceAddr + ":" + servicePort
	server, lis := mock_server.NewGrpcServer(serviceURL, "../../mocks", mock_server.UBUNTU)
	go func() {
		if err := mock_server.RunGrpcServer(server, lis); err != nil {
			log.Println("Failed to run maintenance gRPC server")
		}
	}()

	client := comms.NewClient(serviceURL, &tls.Config{
		RootCAs:            x509.NewCertPool(),
		InsecureSkipVerify: true})
	cli := &comms.Client{}
	cli.MMServiceAddr = serviceAddr
	cli.Transport = grpc.WithTransportCredentials(insecure.NewCredentials())
	comms.WithNetworkDialer(cli.MMServiceAddr)(cli)
	err := client.Connect(context.TODO())
	assert.NoError(f, err)
	return client
}

func Fuzz_PlatformUpdateStatus_FuzzLogic(f *testing.F) {

	client := prepareClientAndServer(f)
	updateController, err := updater.NewUpdateController("", "ubuntu", func() bool { return true })
	assert.NoError(f, err)
	puaScheduler, _ := scheduler.NewPuaScheduler(
		client,
		"test",
		updateController,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			&MockDownloadExecutor{},
			createNullLogger(),
			metadata.NewController(),
		),
		createNullLogger(),
	)
	metaFile, err := os.CreateTemp("/tmp/", "fuzzing-")
	assert.NoError(f, err)
	defer metaFile.Close()
	metadata.MetaPath = metaFile.Name()

	f.Add(
		"root=UUID=22cdff12-4ac0-4a01-bcf0-dc511e434b83 ro quiet splash vt.handoff=7",
		uint64(time.Now().Unix()+3),
		"*/1",
	)
	f.Add(
		"Types: deb\nURIs: https://files.internal.example.intel.com\nSuites: example\nComponents: release\nSigned-By:\npublic GPG key",
		uint64(time.Now().Unix()),
		"*",
	)

	f.Fuzz(func(t *testing.T, kernelAndApt string, singleSchedule uint64, repeatedSchedule string) {
		handleUpdateRes(&pb.PlatformUpdateStatusResponse{
			UpdateSource: &pb.UpdateSource{
				KernelCommand: kernelAndApt,
				OsRepoUrl:     kernelAndApt,
				CustomRepos:   []string{kernelAndApt},
			},
			UpdateSchedule: &pb.UpdateSchedule{
				SingleSchedule: &pb.SingleSchedule{
					StartSeconds: singleSchedule,
					EndSeconds:   singleSchedule,
				},
				RepeatedSchedules: []*pb.RepeatedSchedule{
					{
						DurationSeconds: uint32(singleSchedule),
						CronMinutes:     repeatedSchedule,
						CronHours:       repeatedSchedule,
						CronDayMonth:    repeatedSchedule,
						CronMonth:       repeatedSchedule,
						CronDayWeek:     repeatedSchedule,
					},
				},
			},
		}, puaScheduler, metadata.Meta{}, downloader.NewDownloader(10*time.Minute, 6*time.Hour, &MockDownloadExecutor{}, createNullLogger(), metadata.NewController()), "ubuntu", "test-fqdn")
	})
}

func Test_kernelArgsValidation(t *testing.T) {
	assert.True(
		t,
		kernelRegexp.MatchString(
			"BOOT_IMAGE=/boot/vmlinuz-5.19.0-42-generic console=ttyS0,115200 root=UUID=22cdff12-4ac0-4a01-bcf0-dc511e434b83 ro quiet splash vt.handoff=7",
		),
	)
	assert.True(t, kernelRegexp.MatchString(""))
	assert.False(t, kernelRegexp.MatchString("1+2"))
	assert.False(t, kernelRegexp.MatchString("$a=b"))
	assert.False(t, kernelRegexp.MatchString(string(rune(0))+"abc"))
}

func Test_main_shouldFailIfPathToConfigNotSet(t *testing.T) {
	testLog, hook := test.NewNullLogger()
	log = testLog.WithField("a", "b")

	testLog.ExitFunc = func(i int) {
		assert.Contains(
			t,
			hook.LastEntry().Message,
			"Unable to initialize configuration. Platform update agent will terminate",
		)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovering from %v", r)
		}
	}()

	main()
}

// nolint
func Test_main_shouldFailIfMetadataFileDoesntExists(t *testing.T) {
	testLog, hook := test.NewNullLogger()
	log = testLog.WithField("a", "b")
	argsBefore := os.Args
	os.Args = append(os.Args, "--config", "../../mocks/configs/platform-update-agent.yaml")
	defer func() {
		os.Args = argsBefore
		if r := recover(); r != nil {
			t.Logf("recovering from %v", r)
		}
	}()

	testLog.ExitFunc = func(i int) {
		assert.Contains(
			t,
			hook.LastEntry().Message,
			"Error initializing metadata: open /var/edge-node/pua/metadata.json: no such file or directory",
		)
		os.Exit(0)
	}

	main()
}

func Test_main_happyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-runnning test in short mode.")
	}

	go mock_server.StartMockServer(mock_server.UBUNTU)

	testLog, hook := test.NewNullLogger()
	log = testLog.WithField("a", "b")
	argsBefore := os.Args
	os.Args = append(os.Args, "--config", "../../mocks/configs/valid-platform-update-agent.yaml")
	defer func() {
		os.Args = argsBefore
	}()
	setLogLevel("debug")
	inbmConfigDirectoryPath := "/tmp"
	if _, err := os.Stat(inbmConfigDirectoryPath + "/secret"); os.IsNotExist(err) {
		err = os.Mkdir(inbmConfigDirectoryPath+"/secret", os.ModePerm)
		assert.NoError(t, err)
	}
	file, err := os.Create("/tmp/secret/.provisioned")
	assert.NoError(t, err)
	defer file.Close()

	go main()

	assert.Eventually(t, func() bool {
		for _, entry := range hook.Entries {
			if strings.Contains(entry.Message, "Checking for new update...") {
				return true
			}
		}
		return false
	}, time.Minute*2, 10*time.Millisecond)
}

// nolint

type mockMMClient struct {
	returnedUpdateSource          *pb.UpdateSource
	returnedUpdateSchedule        *pb.UpdateSchedule
	returnedOsType                pb.PlatformUpdateStatusResponse_OSType
	returnedOsProfileUpdateSource *pb.OSProfileUpdateSource
}

func (m *mockMMClient) PlatformUpdateStatus(
	ctx context.Context,
	in *pb.PlatformUpdateStatusRequest,
	opts ...grpc.CallOption,
) (*pb.PlatformUpdateStatusResponse, error) {
	return &pb.PlatformUpdateStatusResponse{
		UpdateSource:          m.returnedUpdateSource,
		UpdateSchedule:        m.returnedUpdateSchedule,
		OsType:                m.returnedOsType,
		OsProfileUpdateSource: m.returnedOsProfileUpdateSource,
	}, nil
}

func Test_handleEdgeInfrastructureManagerRequest_shouldLogWarningIfOneOfTheFieldsIsEmpty_ubuntu(
	t *testing.T,
) {
	testLog, hook := test.NewNullLogger()
	log = testLog.WithField("a", "b")

	wg := &sync.WaitGroup{}
	wg.Add(1)

	tmpFile, err := os.CreateTemp("/tmp", "test-")
	assert.NoError(t, err)
	defer tmpFile.Close()
	config := &puaConfig.Config{
		TickerInterval: time.Second,
		GUID:           "abcd",
		MetadataPath:   tmpFile.Name(),
	}
	metadata.MetaPath = config.MetadataPath
	err = metadata.InitMetadata()
	assert.NoError(t, err)

	updateResChan := make(chan *pb.PlatformUpdateStatusResponse)
	client := comms.NewClient("aa", nil)
	client.MMClient = &mockMMClient{}

	go handleEdgeInfrastructureManagerRequest(
		wg,
		config,
		context.TODO(),
		client,
		updateResChan,
		"ubuntu",
	)

	assert.Eventually(t, func() bool {
		for _, entry := range hook.AllEntries() {
			if strings.Contains(
				entry.Message,
				"skipping response as it is missing one of the fields",
			) {
				return true
			}
		}
		return false
	}, time.Second*10, time.Millisecond)
}

func Test_handleEdgeInfrastructureManagerRequest_shouldLogWarningIfOneOfTheFieldsIsEmpty_emt(
	t *testing.T,
) {
	testLog, hook := test.NewNullLogger()
	log = testLog.WithField("a", "b")

	wg := &sync.WaitGroup{}
	wg.Add(1)

	tmpFile, err := os.CreateTemp("/tmp", "test-")
	assert.NoError(t, err)
	defer tmpFile.Close()
	config := &puaConfig.Config{
		TickerInterval: time.Second,
		GUID:           "abcd",
		MetadataPath:   tmpFile.Name(),
	}
	metadata.MetaPath = config.MetadataPath
	err = metadata.InitMetadata()
	assert.NoError(t, err)

	updateResChan := make(chan *pb.PlatformUpdateStatusResponse)
	client := comms.NewClient("aa", nil)
	client.MMClient = &mockMMClient{}

	go handleEdgeInfrastructureManagerRequest(
		wg,
		config,
		context.TODO(),
		client,
		updateResChan,
		"emt",
	)

	assert.Eventually(t, func() bool {
		for _, entry := range hook.AllEntries() {
			if strings.Contains(
				entry.Message,
				"skipping response as it is missing one of the fields",
			) {
				return true
			}
		}
		return false
	}, time.Second*10, time.Millisecond)
}

func Test_handleEdgeInfrastructureManagerRequest_shouldSetMetadataFromUpdatedToUpToDate(
	t *testing.T,
) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	tmpFile, err := os.CreateTemp("/tmp", "test-")
	assert.NoError(t, err)
	defer tmpFile.Close()
	config := &puaConfig.Config{
		TickerInterval: time.Second,
		GUID:           "abcd",
		MetadataPath:   tmpFile.Name(),
	}
	metadata.MetaPath = config.MetadataPath
	err = metadata.InitMetadata()
	assert.NoError(t, err)
	err = metadata.SetMetaUpdateStatus(pb.UpdateStatus_STATUS_TYPE_UPDATED)
	assert.NoError(t, err)

	updateResChan := make(chan *pb.PlatformUpdateStatusResponse)
	client := comms.NewClient("aa", nil)
	client.MMClient = &mockMMClient{}

	status, err := metadata.GetMetaUpdateStatus()
	assert.NoError(t, err)
	assert.Equal(t, status, pb.UpdateStatus_STATUS_TYPE_UPDATED)

	go handleEdgeInfrastructureManagerRequest(
		wg,
		config,
		context.TODO(),
		client,
		updateResChan,
		"ubuntu",
	)

	assert.Eventually(t, func() bool {
		status, err := metadata.GetMetaUpdateStatus()
		if err != nil {
			return false
		}
		return status == pb.UpdateStatus_STATUS_TYPE_UP_TO_DATE
	}, time.Second*10, time.Millisecond)
}

func Test_handleEdgeInfrastructureManagerRequest_ifResponseIsCorrectThenItShouldBePassedToNextChannel_emt(
	t *testing.T,
) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	tmpFile, err := os.CreateTemp("/tmp", "test-")
	assert.NoError(t, err)
	defer tmpFile.Close()
	config := &puaConfig.Config{
		TickerInterval: time.Second,
		GUID:           "abcd",
		MetadataPath:   tmpFile.Name(),
	}
	metadata.MetaPath = config.MetadataPath
	err = metadata.InitMetadata()
	assert.NoError(t, err)

	updateResChan := make(chan *pb.PlatformUpdateStatusResponse)
	client := comms.NewClient("aa", nil)
	osProfileUpdateSource := &pb.OSProfileUpdateSource{
		OsImageUrl:     "abcd",
		OsImageId:      "abcd",
		OsImageSha:     "abcd",
		ProfileName:    "abcd",
		ProfileVersion: "abcd",
	}
	updateSchedule := &pb.UpdateSchedule{
		SingleSchedule:   nil,
		RepeatedSchedule: nil,
	}
	osType := pb.PlatformUpdateStatusResponse_OS_TYPE_IMMUTABLE
	client.MMClient = &mockMMClient{
		returnedOsProfileUpdateSource: osProfileUpdateSource,
		returnedUpdateSchedule:        updateSchedule,
		returnedOsType:                osType,
	}

	go handleEdgeInfrastructureManagerRequest(
		wg,
		config,
		context.TODO(),
		client,
		updateResChan,
		"emt",
	)

	updateRes := <-updateResChan
	assert.Equal(t, updateSchedule, updateRes.GetUpdateSchedule())
	assert.Equal(t, osProfileUpdateSource, updateRes.GetOsProfileUpdateSource())
	assert.Equal(t, osType, updateRes.GetOsType())
}

func Test_handleEdgeInfrastructureManagerRequest_ifResponseIsCorrectThenItShouldBePassedToNextChannel_ubuntu(
	t *testing.T,
) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	tmpFile, err := os.CreateTemp("/tmp", "test-")
	assert.NoError(t, err)
	defer tmpFile.Close()
	config := &puaConfig.Config{
		TickerInterval: time.Second,
		GUID:           "abcd",
		MetadataPath:   tmpFile.Name(),
	}
	metadata.MetaPath = config.MetadataPath
	err = metadata.InitMetadata()
	assert.NoError(t, err)

	updateResChan := make(chan *pb.PlatformUpdateStatusResponse)
	client := comms.NewClient("aa", nil)
	updateSource := &pb.UpdateSource{
		KernelCommand: "abcd",
		OsRepoUrl:     "",
		CustomRepos:   nil,
	}
	updateSchedule := &pb.UpdateSchedule{
		SingleSchedule:   nil,
		RepeatedSchedule: nil,
	}
	client.MMClient = &mockMMClient{
		returnedUpdateSchedule: updateSchedule,
		returnedUpdateSource:   updateSource,
	}

	go handleEdgeInfrastructureManagerRequest(
		wg,
		config,
		context.TODO(),
		client,
		updateResChan,
		"ubuntu",
	)

	select {
	case updateRes := <-updateResChan:
		assert.Equal(t, updateSchedule, updateRes.GetUpdateSchedule())
		assert.Equal(t, updateSource, updateRes.GetUpdateSource())
	case <-time.After(5 * time.Second):
		t.Fatal("Test failed: Timeout of 5 seconds exceeded")
	}
}

func Test_setLogLevel(t *testing.T) {
	assert.Equal(t, log.Logger.Level, logrus.InfoLevel)

	setLogLevel("dEbuG")
	assert.Equal(t, log.Logger.Level, logrus.DebugLevel)

	setLogLevel("ERROR")
	assert.Equal(t, log.Logger.Level, logrus.ErrorLevel)

	setLogLevel("info")
	assert.Equal(t, log.Logger.Level, logrus.InfoLevel)
}

func Test_continueUpdateAfterOsReboot_emt_shouldUpdateMetadataOnSuccess(t *testing.T) {
	// Setup
	updateController := &updater.UpdateController{}
	cleaner := updater.NewCleaner(
		utils.NewExecutor[[]string](
			func(name string, args ...string) *[]string {
				return new([]string)
			}, func(in *[]string) ([]byte, error) {
				return nil, nil
			}), "emt")
	metadataFile, err := os.CreateTemp("/tmp", "test-metadata-")
	assert.NoError(t, err)
	defer metadataFile.Close()

	inbcFile, err := os.CreateTemp("/tmp", "test-inbc-")
	assert.NoError(t, err)
	_, err = inbcFile.WriteString(`{
    "Status": "SUCCESS",
    "Type": "Software Update",
    "Time": "2023-05-24T15:30:00Z",
    "Metadata": "Additional information about the update",
    "Version": "1.2.3"
}`)
	assert.NoError(t, err)
	inbcFile.Close()

	inbcGranularFile, err := os.CreateTemp("/tmp", "test-inbc-granular-")
	assert.NoError(t, err)
	_, err = inbcGranularFile.WriteString("test granular log")
	assert.NoError(t, err)
	inbcGranularFile.Close()

	conf := &puaConfig.Config{
		INBCLogsPath:         inbcFile.Name(),
		INBCGranularLogsPath: inbcGranularFile.Name(),
		MetadataPath:         metadataFile.Name(),
	}

	metadata.MetaPath = metadataFile.Name()
	err = metadata.InitMetadata()
	assert.NoError(t, err)

	// Set initial metadata
	err = metadata.SetMetaUpdateStatus(pb.UpdateStatus_STATUS_TYPE_UPDATED)
	assert.NoError(t, err)

	desiredProfile := &pb.OSProfileUpdateSource{
		OsImageUrl:     "http://example.com/image.iso",
		OsImageId:      "test-image-id",
		OsImageSha:     "test-image-sha256",
		ProfileName:    "test-profile",
		ProfileVersion: "1.0.0",
	}
	err = metadata.SetMetaOSProfileUpdateSourceDesired(desiredProfile)
	assert.NoError(t, err)

	startActualProfile := &pb.OSProfileUpdateSource{
		OsImageUrl:     "http://example.com/image-old.iso",
		OsImageId:      "test-image-id-old",
		OsImageSha:     "test-image-sha256-old",
		ProfileName:    "test-profile-old",
		ProfileVersion: "0.0.9",
	}
	err = metadata.SetMetaOSProfileUpdateSourceActual(startActualProfile)
	assert.NoError(t, err)

	// Execute
	continueUpdateAfterOsReboot(updateController, conf, cleaner, "emt")

	// Assert
	status, err := metadata.GetMetaUpdateStatus()
	assert.NoError(t, err)
	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_UPDATED, status)

	inProgress, err := metadata.GetMetaUpdateInProgress()
	assert.NoError(t, err)
	assert.Equal(t, string(metadata.NONE), inProgress)

	log, err := metadata.GetMetaUpdateLog()
	assert.NoError(t, err)
	assert.Equal(t, "test granular log", log)

	endActualProfile, err := metadata.GetMetaOSProfileUpdateSourceActual()
	assert.NoError(t, err)
	assert.Equal(t, desiredProfile, endActualProfile)
}

func Test_continueUpdateAfterOsReboot_ifUpdateSucceededThenShouldSetUpdateToNone(t *testing.T) {
	updateController := &updater.UpdateController{}
	cleaner := updater.NewCleaner(
		utils.NewExecutor[[]string](
			func(name string, args ...string) *[]string {
				return new([]string)
			}, func(in *[]string) ([]byte, error) {
				return nil, nil
			}), "ubuntu")
	metadataFile, err := os.CreateTemp("/tmp", "test-metadata-")
	assert.NoError(t, err)
	defer metadataFile.Close()
	inbcFile, err := os.CreateTemp("/tmp", "test-inbc-")
	assert.NoError(t, err)
	defer inbcFile.Close()
	conf := &puaConfig.Config{
		INBCLogsPath: inbcFile.Name(),
		MetadataPath: metadataFile.Name(),
	}

	statusString := `{"Status": "SUCCESS", "Type": "sota", "Time": "2023-07-18 10:56:36", "Metadata": "<?xml version=\"1.0\" encoding=\"utf-8\"?><manifest><type>ota</type><ota><header><type>sota</type><repo>remote</repo></header><type><sota><cmd logtofile=\"y\">update</cmd><mode>no-download</mode><deviceReboot>yes</deviceReboot></sota></type></ota></manifest>", "Error": "", "Version": "v1"}`
	err = os.WriteFile(inbcFile.Name(), []byte(statusString), 0600)
	assert.NoError(t, err)

	metadata.MetaPath = metadataFile.Name()
	err = metadata.InitMetadata()
	assert.NoError(t, err)
	err = metadata.SetMetaUpdateStatus(pb.UpdateStatus_STATUS_TYPE_UPDATED)
	assert.NoError(t, err)
	err = metadata.SetMetaUpdateInProgress(metadata.INBM)
	assert.NoError(t, err)

	continueUpdateAfterOsReboot(updateController, conf, cleaner, "ubuntu")

	status, err := metadata.GetMetaUpdateInProgress()
	assert.NoError(t, err)
	assert.Equal(t, string(metadata.NONE), status)
}

func Test_continueUpdateAfterOsReboot_ShouldFailIfInbcLogsDoNoExist(t *testing.T) {
	testLog, hook := test.NewNullLogger()
	log = testLog.WithField("a", "b")

	testLog.ExitFunc = func(i int) {
		assert.Contains(
			t,
			hook.LastEntry().Message,
			"Update verification failed: reading INBC logs failed: open /tmp/inbc-file.txt: no such file or directory.",
		)
	}

	updateController := &updater.UpdateController{}
	conf := &puaConfig.Config{
		INBCLogsPath: "/tmp/inbc-file.txt",
	}
	cleaner := updater.NewCleaner(
		utils.NewExecutor[[]string](
			func(name string, args ...string) *[]string {
				return new([]string)
			}, func(in *[]string) ([]byte, error) {
				return nil, nil
			}), "ubuntu")

	continueUpdateAfterOsReboot(updateController, conf, cleaner, "ubuntu")
}

func Test_handleUpdateRes_shouldLogWarnIfMetadataFileDoesntExists(t *testing.T) {
	testLog, hook := test.NewNullLogger()
	log = testLog.WithField("a", "b")
	metadata.MetaPath = "/tmp/metadata-test-that-doesnt-exists.json"
	var client *comms.Client
	downloader := downloader.NewDownloader(
		10*time.Minute,
		6*time.Hour,
		&MockDownloadExecutor{},
		createNullLogger(),
		metadata.NewController(),
	)
	puaScheduler, _ := scheduler.NewPuaScheduler(client,
		"abcd",
		scheduler.DoNothingUpdater{},
		downloader,
		createNullLogger(),
	)
	testLog.SetLevel(logrus.WarnLevel)

	handleUpdateRes(&pb.PlatformUpdateStatusResponse{},
		puaScheduler,
		metadata.Meta{},
		downloader,
		"ubuntu",
		"test-fqdn")

	assert.Equal(
		t,
		"failed to update metadata - open /tmp/metadata-test-that-doesnt-exists.json: no such file or directory",
		hook.LastEntry().Message,
	)
}

func Test_handleUbuntuResponse_Immutable(t *testing.T) {
	testLog, hook := test.NewNullLogger()
	log = testLog.WithField("test", "handleUbuntuResponse_Immutable")

	tmpFile, err := os.CreateTemp("", "metadata-")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	metadata.MetaPath = tmpFile.Name()
	err = metadata.InitMetadata()
	assert.NoError(t, err)

	response := &pb.PlatformUpdateStatusResponse{
		OsType: pb.PlatformUpdateStatusResponse_OS_TYPE_IMMUTABLE,
		UpdateSource: &pb.UpdateSource{
			KernelCommand: "valid-kernel-command",
		},
		UpdateSchedule: &pb.UpdateSchedule{
			SingleSchedule: &pb.SingleSchedule{
				StartSeconds: 123456789,
				EndSeconds:   123456789,
			},
		},
	}

	updateResChan := make(chan *pb.PlatformUpdateStatusResponse, 1)

	handleUbuntuResponse(response, updateResChan)

	assert.Contains(t, hook.LastEntry().Message, "skipping IMMUTABLE update request on Ubuntu node")
	assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)

	metaLog, err := metadata.GetMetaUpdateLog()
	assert.NoError(t, err)
	assert.Equal(t, "skipping IMMUTABLE update request on Ubuntu node", metaLog)

	select {
	case res := <-updateResChan:
		t.Errorf("Expected no update to be sent, but got: %+v", res)
	default:
	}
}

func Test_handleEmtResponse_Mutable(t *testing.T) {
	testLog, hook := test.NewNullLogger()
	log = testLog.WithField("test", "handleEmtResponse_Mutable")

	tmpFile, err := os.CreateTemp("", "metadata-")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name()) // Clean up

	metadata.MetaPath = tmpFile.Name()
	err = metadata.InitMetadata()
	assert.NoError(t, err)

	response := &pb.PlatformUpdateStatusResponse{
		OsType: pb.PlatformUpdateStatusResponse_OS_TYPE_MUTABLE,
		OsProfileUpdateSource: &pb.OSProfileUpdateSource{
			OsImageUrl:     "http://example.com/osimage",
			OsImageId:      "osimage-id",
			OsImageSha:     "osimage-sha",
			ProfileName:    "profile-name",
			ProfileVersion: "1.0.0",
		},
		UpdateSchedule: &pb.UpdateSchedule{
			SingleSchedule: &pb.SingleSchedule{
				StartSeconds: 987654321,
				EndSeconds:   987654321,
			},
		},
	}

	updateResChan := make(chan *pb.PlatformUpdateStatusResponse, 1)

	handleEmtResponse(response, updateResChan)

	assert.Contains(t, hook.LastEntry().Message, "skipping MUTABLE update request on Edge Microvisor Toolkit node")
	assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)

	metaLog, err := metadata.GetMetaUpdateLog()
	assert.NoError(t, err)
	assert.Equal(t, "skipping MUTABLE update request on Edge Microvisor Toolkit node", metaLog)

	select {
	case res := <-updateResChan:
		t.Errorf("Expected no update to be sent, but got: %+v", res)
	default:
	}
}

func Test_continueUpdateAfterPuaRestart_UselessTestForCoverage(t *testing.T) {
	metaFile, err := os.CreateTemp("/tmp", "metadata-")
	assert.NoError(t, err)
	defer metaFile.Close()
	metadata.MetaPath = metaFile.Name()
	err = metadata.InitMetadata()
	assert.NoError(t, err)
	controller, err := updater.NewUpdateController("", "ubuntu", func() bool { return true })
	assert.NoError(t, err)

	continueUpdateAfterPuaRestart(controller)
}
