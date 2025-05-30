// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package system

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/testutil"
)

var (
	expectedTimezone = "CET"

	expectedLocale = model.Locale{
		CountryName: "United States",
		CountryAbbr: "US",
		LangName:    "English",
		LangAbbr:    "en",
	}

	expectedMachineHardwareName = "x86_64"
	expectedKernelName          = "Linux"
	expectedKernelRelease       = "6.12.20-1.emt3"
	expectedKernelVersion       = "#1 SMP PREEMPT_DYNAMIC Fri Mar 28 05:28:06 UTC 2025"
	expectedOperatingSystem     = "GNU/Linux"

	expectedRelease = model.Release{
		ID:           "ubuntu",
		VersionID:    "24.10",
		Version:      "24.10 (Oracular Oriole)",
		Codename:     "oracular",
		Family:       "debian",
		BuildID:      "20240601",
		ImageID:      "ubuntu-image",
		ImageVersion: "1.0.0",
	}

	expectedUptime       = 12345.67
	expectedSerialNumber = "SN123456789"
	expectedSystemUUID   = "UUID-1234-5678-ABCD"
)

func TestGetTimezoneSuccess(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("date", []string{"+%Z"}, []byte(expectedTimezone), nil)

	timezone, err := GetTimezone(testutil.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, expectedTimezone, timezone)
}

func TestGetTimezoneFailure(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("date", []string{"+%Z"}, nil, os.ErrPermission)

	_, err := GetTimezone(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to get timezone")
}

func TestGetLocaleDataSuccess(t *testing.T) {
	testutil.ClearMockOutputs()
	localeStr := fmt.Sprintf(`country_name="%s"
country_ab2="%s"
lang_name="%s"
lang_ab="%s"
`, expectedLocale.CountryName, expectedLocale.CountryAbbr, expectedLocale.LangName, expectedLocale.LangAbbr)
	testutil.SetMockOutput("locale", []string{"-k", "LC_ADDRESS"}, []byte(localeStr), nil)

	locale, err := GetLocaleData(testutil.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, expectedLocale, locale)
}

func TestGetLocaleDataFailure(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("locale", []string{"-k", "LC_ADDRESS"}, nil, os.ErrPermission)

	_, err := GetLocaleData(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to get locale data")
}

func TestGetKernelDataSuccess(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("uname", []string{"-m"}, []byte(expectedMachineHardwareName), nil)
	testutil.SetMockOutput("uname", []string{"-s"}, []byte(expectedKernelName), nil)
	testutil.SetMockOutput("uname", []string{"-r"}, []byte(expectedKernelRelease), nil)
	testutil.SetMockOutput("uname", []string{"-v"}, []byte(expectedKernelVersion), nil)
	testutil.SetMockOutput("uname", []string{"-o"}, []byte(expectedOperatingSystem), nil)

	expectedKernel := model.Kernel{
		Machine: expectedMachineHardwareName,
		Name:    expectedKernelName,
		Release: expectedKernelRelease,
		Version: expectedKernelVersion,
		System:  expectedOperatingSystem,
	}

	kernel, err := GetKernelData(testutil.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, expectedKernel, kernel)
}

func TestGetKernelDataMachineHardwareNameFailure(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("uname", []string{"-m"}, nil, os.ErrPermission)

	_, err := GetKernelData(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to get OS information (machine hardware name)")
}

func TestGetKernelDataKernelNameFailure(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("uname", []string{"-m"}, []byte(expectedMachineHardwareName), nil)
	testutil.SetMockOutput("uname", []string{"-s"}, nil, os.ErrPermission)

	_, err := GetKernelData(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to get OS information (kernel name)")
}

func TestGetKernelDataKernelReleaseFailure(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("uname", []string{"-m"}, []byte(expectedMachineHardwareName), nil)
	testutil.SetMockOutput("uname", []string{"-s"}, []byte(expectedKernelName), nil)
	testutil.SetMockOutput("uname", []string{"-r"}, nil, os.ErrPermission)

	_, err := GetKernelData(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to get OS information (kernel release)")
}

func TestGetKernelDataKernelVersionFailure(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("uname", []string{"-m"}, []byte(expectedMachineHardwareName), nil)
	testutil.SetMockOutput("uname", []string{"-s"}, []byte(expectedKernelName), nil)
	testutil.SetMockOutput("uname", []string{"-r"}, []byte(expectedKernelRelease), nil)
	testutil.SetMockOutput("uname", []string{"-v"}, nil, os.ErrPermission)

	_, err := GetKernelData(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to get OS information (kernel version)")
}

func TestGetKernelDataOperatingSystemFailure(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("uname", []string{"-m"}, []byte(expectedMachineHardwareName), nil)
	testutil.SetMockOutput("uname", []string{"-s"}, []byte(expectedKernelName), nil)
	testutil.SetMockOutput("uname", []string{"-r"}, []byte(expectedKernelRelease), nil)
	testutil.SetMockOutput("uname", []string{"-v"}, []byte(expectedKernelVersion), nil)
	testutil.SetMockOutput("uname", []string{"-o"}, nil, os.ErrPermission)

	_, err := GetKernelData(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to get OS information (operating system)")
}

func TestGetReleaseDataSuccess(t *testing.T) {
	testutil.ClearMockOutputs()
	expectedReleaseString := fmt.Sprintf(`ID=%s
VERSION_ID="%s"
VERSION="%s"
VERSION_CODENAME= %s 
ID_LIKE=%s
BUILD_ID=%s
IMAGE_ID=%s
IMAGE_VERSION=%s
`, expectedRelease.ID, expectedRelease.VersionID, expectedRelease.Version, expectedRelease.Codename, expectedRelease.Family,
		expectedRelease.BuildID, expectedRelease.ImageID, expectedRelease.ImageVersion)
	testutil.SetMockOutput("cat", []string{"/etc/os-release"}, []byte(expectedReleaseString), nil)

	release, err := GetReleaseData(testutil.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, expectedRelease, release)
}

func TestGetReleaseDataFailure(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("cat", []string{"/etc/os-release"}, nil, os.ErrPermission)

	_, err := GetReleaseData(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to read data from /etc/os-release")
}

func TestGetUptimeDataSuccess(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("cat", []string{"/proc/uptime"}, []byte(fmt.Sprintf("%.2f 99999.99\n", expectedUptime)), nil)

	uptime, err := GetUptimeData(testutil.TestCmdExecutor)
	require.NoError(t, err)
	require.InDelta(t, expectedUptime, uptime, 0.01)
}

func TestGetUptimeDataFailure(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("cat", []string{"/proc/uptime"}, nil, os.ErrPermission)

	_, err := GetUptimeData(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to read data from /proc/uptime")
}

func TestGetUptimeDataEmptyOutput(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("cat", []string{"/proc/uptime"}, []byte("\n"), nil)

	_, err := GetUptimeData(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "unexpected format in /proc/uptime")
}

func TestGetUptimeDataMalformedOutput(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("cat", []string{"/proc/uptime"}, []byte("not_a_number something_else\n"), nil)

	_, err := GetUptimeData(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to parse uptime value")
}

func TestGetSerialNumberSuccess(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("sudo", []string{"dmidecode", "-s", "system-serial-number"}, []byte(expectedSerialNumber), nil)

	serial, err := GetSerialNumber(testutil.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, expectedSerialNumber, serial)
}

func TestGetSerialNumberFailure(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("sudo", []string{"dmidecode", "-s", "system-serial-number"}, nil, os.ErrPermission)

	_, err := GetSerialNumber(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to get serial number")
}

func TestGetSystemUUIDSuccess(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("sudo", []string{"dmidecode", "-s", "system-uuid"}, []byte(expectedSystemUUID), nil)

	uuid, err := GetSystemUUID(testutil.TestCmdExecutor)
	require.NoError(t, err)
	require.Equal(t, expectedSystemUUID, uuid)
}

func TestGetSystemUUIDFailure(t *testing.T) {
	testutil.ClearMockOutputs()
	testutil.SetMockOutput("sudo", []string{"dmidecode", "-s", "system-uuid"}, nil, os.ErrPermission)

	_, err := GetSystemUUID(testutil.TestCmdExecutor)
	require.ErrorContains(t, err, "failed to get system UUID")
}
