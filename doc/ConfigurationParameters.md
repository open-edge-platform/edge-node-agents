# Intel Manageability Configuration File

## Overview

The `intel_manageability.conf` file is a JSON-based configuration file used to define settings for Intel's in-band manageability features. This file is primarily used to configure the OS updater and other related components.

---

## Configuration Structure

The configuration file is structured as follows:

```json
{
    "os_updater": {
        "trustedRepositories": [],
        "proceedWithoutRollback": true
    },
    "luks": {
        "volumePath": "/var/intel-manageability/secret.img",
        "mapperName": "intel-manageability-secret",
        "mountPoint": "/etc/intel-manageability/secret",
        "passwordLength": 32,
        "size": 32,
        "useTPM": true,
        "user": "root",
        "group": "root"
    }
}
```

### **Fields**

#### `os_updater`

- **Description:** Contains configuration settings for the OS updater.
- **Type:** Object
- **Required:** Yes

##### **`trustedRepositories`**

- **Description:** A list of trusted repository URLs that the OS updater can use for updates.
- **Type:** Array of strings
- **Required:** Yes
- **Default Value:** An empty array (`[]`)

##### **`proceedWithoutRollback`**

- **Description:** Indicates whether the OS updater should proceed with updates even if rollback functionality is unavailable.
- **Type:** Boolean
- **Required:** Yes
- **Default Value:** `true`

#### `luks`
- **Description:** Contains configuration settings for LUKS (Linux Unified Key Setup) encryption used for secure storage of certificates and keys.
- **Type:** Object
- **Required:** Yes

##### **`volumePath`**
- **Description:** Path to the LUKS encrypted volume file.
- **Type:** String
- **Required:** Yes
- **Default Value:** `"/var/intel-manageability/secret.img"`

##### **`mapperName`**
- **Description:** The device mapper name for the encrypted volume.
- **Type:** String
- **Required:** Yes
- **Default Value:** `"intel-manageability-secret"`

##### **`mountPoint`**
- **Description:** The directory where the encrypted volume will be mounted. This path is used for storing TLS certificates and keys.
- **Type:** String
- **Required:** Yes
- **Default Value:** `"/etc/intel-manageability/secret"`

##### **`passwordLength`**
- **Description:** The length of the password used for LUKS encryption (in bytes).
- **Type:** Integer
- **Required:** Yes
- **Default Value:** `32`

##### **`size`**
- **Description:** The size of the LUKS encrypted volume (in MB).
- **Type:** Integer
- **Required:** Yes
- **Default Value:** `32`

##### **`useTPM`**
- **Description:** Indicates whether TPM (Trusted Platform Module) should be used for key management.
- **Type:** Boolean
- **Required:** Yes
- **Default Value:** `true`

##### **`user`**
- **Description:** The user owner of the mounted encrypted volume.
- **Type:** String
- **Required:** Yes
- **Default Value:** `"root"`

##### **`group`**
- **Description:** The group owner of the mounted encrypted volume.
- **Type:** String
- **Required:** Yes
- **Default Value:** `"root"`
---

## Example Configuration

Here is an example of a valid configuration file with trusted repositories and LUKS encryption:

```json
{
    "os_updater": {
        "trustedRepositories": [
            "https://repo1.example.com",
            "https://repo2.example.com"
        ],
        "proceedWithoutRollback": false
    },
    "luks": {
        "volumePath": "/var/intel-manageability/secret.img",
        "mapperName": "intel-manageability-secret",
        "mountPoint": "/etc/intel-manageability/secret",
        "passwordLength": 32,
        "size": 32,
        "useTPM": true,
        "user": "root",
        "group": "root"
    }
}
```

---

## Notes

- Ensure that all URLs in the `trustedRepositories` array are valid and accessible.
- The `proceedWithoutRollback` field determines whether updates should proceed if rollback functionality is unavailable. Set this to `false` if rollback is critical for your environment.
- The configuration file must conform to the JSON schema used for validation.
- If the `trustedRepositories` array is empty, the OS updater will not have any repositories to use for updates.
- The LUKS configuration is used for secure storage of TLS certificates and keys. The `mountPoint` directory path is dynamically used by the TLS certificate generation system.
- When `useTPM` is set to `true`, the system will use the Trusted Platform Module for enhanced security of encryption keys.
- The `volumePath` should point to a location with sufficient disk space for the encrypted volume.
- Ensure that the `user` and `group` specified in the LUKS configuration have appropriate permissions for the mount point.

---

## Validation

To validate the configuration file, use the JSON schema provided in the project. You can use tools like `gojsonschema` or online validators to ensure the file is valid.

---

## Location

The configuration file is located at:

```cmd
/etc/intel_manageability.conf
```

---

## Troubleshooting

### OS Updater Issues

- **Issue:** The OS updater fails to fetch updates.
  - **Solution:** Ensure that the URLs in `trustedRepositories` are correct and accessible.

- **Issue:** Validation errors when saving the configuration file.
  - **Solution:** Verify the file against the JSON schema to ensure it conforms to the expected structure.

- **Issue:** Updates fail due to rollback functionality being unavailable.
  - **Solution:** Set `proceedWithoutRollback` to `true` to allow updates to proceed without rollback.

### LUKS Configuration Issues

- **Issue:** TLS certificate generation fails with "permission denied" errors.
  - **Solution:** Ensure the LUKS `mountPoint` directory exists and has proper permissions for the specified `user` and `group`.

- **Issue:** LUKS volume fails to mount on system startup.
  - **Solution:** Verify that the `volumePath` exists and the system has the necessary LUKS utilities installed. Check that TPM is properly configured if `useTPM` is set to `true`.

- **Issue:** Insufficient disk space for LUKS volume.
  - **Solution:** Increase the `size` parameter in the LUKS configuration or ensure sufficient disk space is available at the `volumePath` location.

- **Issue:** TPM-related errors when `useTPM` is enabled.
  - **Solution:** Ensure TPM 2.0 is available and properly configured on the system. Check that tpm2-tools are installed and the TPM is not locked or owned by another process.

---

## LUKS and TLS Certificate Integration

The LUKS configuration plays a crucial role in securing TLS certificates and keys used by the Intel Manageability system. The `mountPoint` specified in the LUKS configuration is dynamically used by the TLS certificate generation system to store:

- CA (Certificate Authority) certificates and private keys
- Server certificates for inbd daemon
- Client certificates for inbc client

This integration ensures that all cryptographic material is stored in an encrypted volume, providing enhanced security for sensitive data at rest.

### Certificate Storage Locations

When LUKS is configured, the following certificate files are stored in the encrypted mount point:

- `{mountPoint}/ca.crt` - Certificate Authority certificate
- `{mountPoint}/ca.key` - Certificate Authority private key
- `{mountPoint}/inbd.crt` - Server certificate for inbd daemon
- `{mountPoint}/inbd.key` - Server private key for inbd daemon
- `{mountPoint}/inbc.crt` - Client certificate for inbc client
- `{mountPoint}/inbc.key` - Client private key for inbc client

If the LUKS configuration is not available or the mount point is not specified, the system falls back to the default location: `/etc/intel-manageability/secret`.

---

## License

This configuration file is part of the Intel Manageability project and is licensed under the Apache-2.0 License.
