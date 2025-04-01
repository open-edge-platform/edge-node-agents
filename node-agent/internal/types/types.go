// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package types

type Credentials struct {
	PrivateKey     []byte
	CertSigningReq []byte
	Certificate    []byte
}
