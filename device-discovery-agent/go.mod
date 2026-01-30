// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

module device-discovery

// remains with Go 1.24.1 till EMT Go is updated to support Go 1.24.4
go 1.24.9

require (
	github.com/open-edge-platform/edge-node-agents/common v1.9.1
	github.com/open-edge-platform/infra-onboarding/onboarding-manager v1.33.0
	golang.org/x/oauth2 v0.33.0
	google.golang.org/grpc v1.77.0
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.10-20250912141014-52f32327d4b0.1 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	golang.org/x/net v0.46.1-0.20251013234738-63d1a5100f82 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251022142026-3a174f9686a8 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)
