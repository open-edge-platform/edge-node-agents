package appsource

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/intel/intel-inb-manageability/internal/inbd/utils"
	pb "github.com/intel/intel-inb-manageability/pkg/api/inbd/v1"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
    fs := afero.NewMemMapFs() // Use an in-memory filesystem for testing

    tests := []struct {
        name           string
        adder          *Adder
        req            *pb.AddApplicationSourceRequest
        expectedError  string
        expectedOutput string
    }{
        {
            name: "Success",
            adder: &Adder{
                openFileFunc: func(fs afero.Fs, name string, flag int, perm os.FileMode) (afero.File, error) {
                    return fs.OpenFile(name, flag, perm)
                },
                loadConfigFunc: func(fs afero.Fs, path string) (*utils.Configurations, error) {
                    return &utils.Configurations{
                        OSUpdater: struct {
                            TrustedRepositories []string `json:"trustedRepositories"`
                        }{
                            TrustedRepositories: []string{
                                "https://example.com/repo1",
                                "https://example.com/repo2",
                            },
                        },
                    }, nil
                },
                isTrustedRepoFunc: func(uri string, config *utils.Configurations) bool {
                    return true
                },
                addGpgKeyFunc: func(uri, name string, requestCreator func(string, string, io.Reader) (*http.Request, error), client *http.Client, executor utils.Executor) error {
                    return nil
                },
                fs: fs,
            },
            req: &pb.AddApplicationSourceRequest{
                GpgKeyUri:  "http://example.com/key.asc",
                GpgKeyName: "example-key.gpg",
                Filename:   "example.list",
                Source:     []string{"deb http://example.com/ubuntu focal main"},
            },
            expectedError:  "",
            expectedOutput: "deb http://example.com/ubuntu focal main\n",
        },
        {
            name: "ConfigurationLoadFailure",
            adder: &Adder{
                loadConfigFunc: func(fs afero.Fs, path string) (*utils.Configurations, error) {
                    return nil, os.ErrNotExist
                },
                fs: fs,
            },
            req: &pb.AddApplicationSourceRequest{
                GpgKeyUri:  "http://example.com/key.asc",
                GpgKeyName: "example-key.gpg",
                Filename:   "example.list",
                Source:     []string{"deb http://example.com/ubuntu focal main"},
            },
            expectedError: "error loading config",
        },
        {
            name: "GpgKeyVerificationFailure",
            adder: &Adder{
                loadConfigFunc: func(fs afero.Fs, path string) (*utils.Configurations, error) {
                    return &utils.Configurations{
                        OSUpdater: struct {
                            TrustedRepositories []string `json:"trustedRepositories"`
                        }{
                            TrustedRepositories: []string{
                                "https://example.com/repo1",
                                "https://example.com/repo2",
                            },
                        },
                    }, nil
                },
                isTrustedRepoFunc: func(uri string, config *utils.Configurations) bool {
                    return false
                },
                fs: fs,
            },
            req: &pb.AddApplicationSourceRequest{
                GpgKeyUri:  "http://example.com/key.asc",
                GpgKeyName: "example-key.gpg",
                Filename:   "example.list",
                Source:     []string{"deb http://example.com/ubuntu focal main"},
            },
            expectedError: "GPG key URI verification failed",
        },
        {
            name: "GpgKeyAdditionFailure",
            adder: &Adder{
                loadConfigFunc: func(fs afero.Fs, path string) (*utils.Configurations, error) {
                    return &utils.Configurations{
                        OSUpdater: struct {
                            TrustedRepositories []string `json:"trustedRepositories"`
                        }{
                            TrustedRepositories: []string{
                                "https://example.com/repo1",
                                "https://example.com/repo2",
                            },
                        },
                    }, nil
                },
                isTrustedRepoFunc: func(uri string, config *utils.Configurations) bool {
                    return true
                },
                addGpgKeyFunc: func(uri, name string, requestCreator func(string, string, io.Reader) (*http.Request, error), client *http.Client, executor utils.Executor) error {
                    return os.ErrPermission
                },
                fs: fs,
            },
            req: &pb.AddApplicationSourceRequest{
                GpgKeyUri:  "http://example.com/key.asc",
                GpgKeyName: "example-key.gpg",
                Filename:   "example.list",
                Source:     []string{"deb http://example.com/ubuntu focal main"},
            },
            expectedError: "error adding GPG key",
        },
        {
            name: "FileCreationFailure",
            adder: &Adder{
                openFileFunc: func(fs afero.Fs, name string, flag int, perm os.FileMode) (afero.File, error) {
                    return nil, os.ErrPermission
                },
                loadConfigFunc: func(fs afero.Fs, path string) (*utils.Configurations, error) {
                    return &utils.Configurations{
                        OSUpdater: struct {
                            TrustedRepositories []string `json:"trustedRepositories"`
                        }{
                            TrustedRepositories: []string{
                                "https://example.com/repo1",
                                "https://example.com/repo2",
                            },
                        },
                    }, nil
                },
                isTrustedRepoFunc: func(uri string, config *utils.Configurations) bool {
                    return true
                },
                addGpgKeyFunc: func(uri, name string, requestCreator func(string, string, io.Reader) (*http.Request, error), client *http.Client, executor utils.Executor) error {
                    return nil
                },
                fs: fs,
            },
            req: &pb.AddApplicationSourceRequest{
                Filename: "example.list",
                Source:   []string{"deb http://example.com/ubuntu focal main"},
            },
            expectedError: "error opening source list file",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.adder.Add(tt.req)
            if tt.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
            } else {
                assert.NoError(t, err)
                content, err := afero.ReadFile(fs, "/etc/apt/sources.list.d/example.list")
                assert.NoError(t, err)
                assert.Equal(t, tt.expectedOutput, string(content))
            }
        })
    }
}

type mockExecutor struct {
	commands [][]string
	stdout   []string
	stderr   []string
	errors   []error
}

func (m *mockExecutor) Execute(command []string) ([]byte, []byte, error) {
	m.commands = append(m.commands, command)
	var stdout, stderr string
	if len(m.stderr) > 0 {
		stderr = m.stderr[0]
		m.stderr = m.stderr[1:]
	}
	if len(m.stdout) > 0 {
		stdout = m.stdout[0]
		m.stdout = m.stdout[1:]
	}
	return []byte(stdout), []byte(stderr), m.errors[0]
}

func TestAddGpgKey_Success(t *testing.T) {
	mockExec := &mockExecutor{
		stdout: []string{""},
		errors: []error{nil},
	}
	mockClient:= &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("file content")),
			}
		}),
	}
	mockRequestCreator := http.NewRequest

    err := addGpgKey(
        "http://example.com/key.asc",
        "example-key.gpg",
        mockRequestCreator,
        mockClient,
        mockExec,
    )
    assert.NoError(t, err)
}

func TestAddGpgKey_RequestCreationFailure(t *testing.T) {
	mockExec := &mockExecutor{
		stdout: []string{""},
		errors: []error{nil},
	}
	mockClient:= &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("file content")),
			}
		}),
	}

    err := addGpgKey(
        "invalid-url",
        "example-key.gpg",
        func(method, url string, body io.Reader) (*http.Request, error) {
            return nil, fmt.Errorf("mock request creation error")
        },
        mockClient,
        mockExec,
    )
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "error creating request")
}

func TestAddGpgKey_HTTPFailure(t *testing.T) {
	mockExec := &mockExecutor{
		stdout: []string{""},
		errors: []error{nil},
	}
	mockClient := &http.Client{
        Transport: roundTripperFunc(func(req *http.Request) *http.Response {
            return nil // Simulate an HTTP failure
        }),
    }
	mockRequestCreator := http.NewRequest

    err := addGpgKey(
        "http://example.com/key.asc",
        "example-key.gpg",
        mockRequestCreator,
        mockClient,
        mockExec,
    )
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "error performing request")
}

func TestAddGpgKey_Non200StatusCode(t *testing.T) {
	mockExec := &mockExecutor{
		stdout: []string{""},
		errors: []error{nil},
	}
	mockClient:= &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader("")),
			}
		}),
	}

	mockRequestCreator := http.NewRequest

    err := addGpgKey(
        "http://example.com/key.asc",
        "example-key.gpg",
        mockRequestCreator,
        mockClient,
        mockExec,
    )
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "error getting GPG key.  Status code: 400")
}

func TestAddGpgKey_DearmorFailure(t *testing.T) {
	mockExec := &mockExecutor{
		stdout: []string{""},
		errors: []error{errors.New("mock dearmor error")},
	}
	mockClient:= &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("file content")),
			}
		}),
	}
	mockRequestCreator := http.NewRequest

    err := addGpgKey(
        "http://example.com/key.asc",
        "example-key.gpg",
        mockRequestCreator,
        mockClient,
        mockExec,
    )
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "error dearmoring GPG key")
}

// roundTripperFunc is a helper type to mock http.RoundTripper
type roundTripperFunc func(req *http.Request) *http.Response

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}
