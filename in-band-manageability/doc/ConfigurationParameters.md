<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
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

---

## Example Configuration

Here is an example of a valid configuration file with trusted repositories:

```json
{
    "os_updater": {
        "trustedRepositories": [
            "https://repo1.example.com",
            "https://repo2.example.com"
        ],
        "proceedWithoutRollback": false
    }
}
```

---

## Notes

- Ensure that all URLs in the `trustedRepositories` array are valid and accessible.
- The `proceedWithoutRollback` field determines whether updates should proceed if rollback functionality is unavailable. Set this to `false` if rollback is critical for your environment.
- The configuration file must conform to the JSON schema used for validation.
- If the `trustedRepositories` array is empty, the OS updater will not have any repositories to use for updates.

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

- **Issue:** The OS updater fails to fetch updates.
  - **Solution:** Ensure that the URLs in `trustedRepositories` are correct and accessible.

- **Issue:** Validation errors when saving the configuration file.
  - **Solution:** Verify the file against the JSON schema to ensure it conforms to the expected structure.

- **Issue:** Updates fail due to rollback functionality being unavailable.
  - **Solution:** Set `proceedWithoutRollback` to `true` to allow updates to proceed without rollback.

---

## License

This configuration file is part of the Intel Manageability project and is licensed under the Apache-2.0 License.
