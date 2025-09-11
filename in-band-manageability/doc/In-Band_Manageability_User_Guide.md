
<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->

<!--
    Section: Filepath Reference

    This section provides the file path for the In-Band Manageability User Guide markdown file.
    It is intended to help users and developers quickly locate the documentation source within the project directory structure.

    Usage:
    - Reference this section when searching for the user guide file.
    - Useful for onboarding, documentation maintenance, and navigation within the repository.
-->
# In-Band Manageability Framework User Guide&reg;

<details>
<summary>Table of Contents</summary>

1. [FOTA (Firmware Update Over the Air)](#fota-firmware-update-over-the-air)
    1. [How FOTA Uses `firmware_tool_info.conf`](#how-fota-uses-firmware_tool_infoconf)
    2. [Updating `firmware_tool_info.conf` for a New Platform Type](#updating-firmware_tool_infoconf-for-a-new-platform-type)
    3. [Example Entry for a New Platform](#example-entry-for-a-new-platform)
    4. [Notes](#notes)

</details>

## FOTA (Firmware Update Over the Air)

The FOTA (Firmware Over-The-Air) update process uses the `firmware_tool_info.conf` file to determine how firmware updates are performed for different platform types. This configuration file specifies parameters such as platform identifiers, firmware image locations, update strategies, and other platform-specific settings.

### How FOTA Uses `firmware_tool_info.conf`

1. **Platform Identification**: The FOTA update system reads the `firmware_tool_info.conf` file to identify the target platform type. Each section or entry in the file corresponds to a supported platform.
2. **Firmware Image Selection**: The configuration specifies the location or naming convention of the firmware image to be used for the update.
3. **Update Parameters**: Additional parameters such as update method (e.g., full image, delta update), verification steps, and rollback options are defined per platform.
4. **Execution**: During the update process, the FOTA system loads the relevant configuration for the detected platform and performs the update according to the specified parameters.

### Updating `firmware_tool_info.conf` for a New Platform Type

1. **Identify the New Platform**: Determine the unique identifier (e.g., platform name or code) for the new hardware or device type.
2. **Add a New Section**: In the `/etc/firmware_tool_info.conf` file, add a new section or entry for the new platform. Use the appropriate syntax (e.g., INI section headers or key-value pairs).
3. **Specify Required Parameters**:
Add the following required parameters for the new platform, using the structure and fields from `fpm-templates/etc/firmware_tool_info.conf`:

    * **name**: Name of the firmware product (e.g., "Alder Lake Client Platform").
    * **guid**: (Optional) Set to `true` if a GUID is required for the product.
    * **tool_options**: (Optional) Command line parameter switches that may be required by the tool (e.g., "/b /p")
    * **bios_vendor**: BIOS vendor string (e.g., "Intel Corporation").
    * **firmware_tool**: Path or name of the firmware update tool (e.g., "/opt/afulnx/afulnx_64" or "fwupdate").
    * **firmware_tool_args**: (Optional) Arguments to pass to the firmware tool for applying updates (e.g., "--apply").
    * **firmware_tool_check_args**: (Optional) Arguments to check the firmware tool status (e.g., "-s").
    * **firmware_file_type**: Type of the firmware file (e.g., "xx").

    Example:

    ```ini
    name = Alder Lake Client Platform
    guid = true
    tool_options = /b /p
    bios_vendor = Intel Corporation
    firmware_tool = /opt/afulnx/afulnx_64
    firmware_tool_args = --apply
    firmware_tool_check_args = -s
    firmware_file_type = xx
    ```

4. **Save and Deploy**: Save the updated `/etc/firmware_tool_info.conf` file.
5. **Test the Update**: Perform a test FOTA update on the new platform to verify that the configuration is correct and the update process completes successfully.

### Example Entry for a New Platform

```ini
       {
        "name": "<Platform Name>",
        "guid": true,
        "bios_vendor": "<Vendor>",
        "firmware_tool": "<Update Tool used>",
        "firmware_tool_args": "<Tool arguments>",
        "firmware_tool_check_args": "<Tool check argument>",
        "firmware_file_type": "xx"
      },
```

### Notes

* Always back up the original `firmware_tool_info.conf` before making changes.
* Consult the FOTA system documentation for any platform-specific configuration options.
* Ensure that the firmware image specified is compatible with the new platform.
