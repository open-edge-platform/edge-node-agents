<!-- SPDX-FileCopyrightText: (C) 2025 Intel Corporation -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Platform Update Agent (PUA) Testing Guide

This guide explains how to run PUA against a 'mock' maintenance manager. These instructions are for Ubuntu/Debian-based systems. For simplicity, an isolated test machine that can run both the mock maintenance manager and the platform update agent (PUA), and has tools installed to build PUA, is recommended.

## Installing the Certificate

First, you must install the test certificate on the system where PUA will run.

WARNING: Exercise extreme caution when modifying your system's trust store.

1. The `server-cert.pem` and `server-key.pem` in this repository are for testing purposes only. DO NOT use these in a production environment or on any system that handles sensitive information.

2. Adding this certificate to your system's trust store carries significant security risks:
   - It allows anyone with access to this repository to be trusted by your system.
   - It could enable man-in-the-middle attacks and other security vulnerabilities.

3. Recommendations:
   a. For testing: Use these certificates only on isolated, temporary virtual machines or containerized environments dedicated to testing.
   b. For development/production: Generate your own self-signed certificates or obtain properly issued certificates from a trusted Certificate Authority.

4. If you must proceed for testing purposes:
   - Use a dedicated testing environment.
   - Remove the certificate from the trust store immediately after testing.
   - Never use this setup for handling real, sensitive data.

5. To install (on a dedicated test system only):

   a. Copy the certificate to the appropriate trust store directory:
      For Ubuntu/Debian:
      ```
      sudo cp server-cert.pem /usr/local/share/ca-certificates/mm-mock-cert.crt
      ```

   b. Update the CA store:
      For Ubuntu/Debian:
      ```
      sudo update-ca-certificates
      ```

   c. Verify the installation (if this command returns results, the certificate has been successfully installed):
      ```
      ls -l /etc/ssl/certs | grep mm-mock-cert
      ```

   IMPORTANT: Remember to remove this certificate from your trust store immediately after testing:
   ```
   sudo rm /usr/local/share/ca-certificates/mm-mock-cert.crt
   sudo update-ca-certificates --fresh
   ```

Remember: Security best practices strongly discourage using publicly available certificates and keys in any trust store, even for testing purposes, unless in a completely isolated environment.

## Running the mock Maintenance Manager

You can choose `UBUNTU` or `EMT` which will change the contents of the mock Maintenance Manager response to PUA. Example with `EMT`:

```
make mmbuild && ( cd build/artifacts && ./maintenance-mngr-mock -server EMT )
```

## Running PUA

There is an optional `force-os` parameter you can use to make PUA think it is on a particular OS. This can be `emt` or `ubuntu`.  Example with `emt`:

```
make puabuild && build/artifacts/platform-update-agent -config mocks/configs/empty-platform-update-agent.yaml -force-os emt
```

This config assumes PUA will be running on the same machine as the mock maintenance manager.  You can specify a different IP or change other options like debug level by editing `empty-platform-update-agent.yaml` or specifying a different config file.
