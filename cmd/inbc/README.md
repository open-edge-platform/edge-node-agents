# Intel® In-band Manageability Command-line Utility (INBC)

<details>
<summary>Table of Contents</summary>

1. [Introduction](#introduction)
2. [Commands](#commands)
   1. [FOTA](#fota)
   2. [SOTA](#sota)
   3. [Source Application Add](#source-application-add)
   4. [Source Application Remove](#source-application-remove)
   5. [Source OS Update](#source-os-update)
   6. [Configuration Load](#load)
   7. [Configuration Get](#get)
   8. [Configuration Set](#set)
   9. [Configuration Remove](#remove)
   10. [Configuration Append](#append)
   11. [Query](#query)
   12. [Restart](#restart)
   13. [Shutdown](#shutdown)

</details>

## Introduction

Intel® In-Band Manageability command-line utility, INBC, is a software utility running on a host managing an Edge IoT Device.  It allows the user to perform Device Management operations like system update from the command-line. This may be used in lieu of using the cloud update mechanism.

## Commands

### FOTA

#### Description

Performs a Firmware Over The Air (FOTA) update.

#### Usage

```commandline
inbc fota {--uri URI} 
   {--releasedate RELEASE_DATE}
   [--reboot; default=true]
   [--username USERNAME; default=""]
   [--signature SIGNATURE; default=""]
```

#### Examples

```commandline
inbc fota 
   --uri <URI to TAR package>/BIOSUPDATE.tar
   --releasedate 2026-06-01
   --reboot=false
   --signature <hash string of signature>
```

### SOTA

#### Description

Performs a Software Over The Air (SOTA) update.

##### Edge Device

There are two possible software updates on an edge device depending on the Operating System on the device. If the OS is Ubuntu, then the update will be performed using the Ubuntu update mechanism.

System update flow can be broken into two parts:

1. Pre-reboot: The pre-boot part is when a system update is triggered.
2. Post-reboot: The post-boot checks the health of critical manageability services and takes corrective action.

SOTA on Ubuntu and EMT OS is supported in 3 modes:

1. Update/Full - Performs the software update.
2. No download - Retrieves and installs packages.
3. Download only - Retrieve packages (will not unpack or install).

By default, when SOTA is performing an installation, it will upgrade all eligible packages. The user can optionally specify a list of packages to upgrade (or install if not present) via the [--package-list, -p=PACKAGES] option.

#### Usage

```commandline
inbc sota {--uri URI} 
   [--releasedate RELEASE_DATE; default="2026-12-31"] 
   [--mode MODE; default="full", choices=["full","no-download", "download-only"] ]
   [--reboot; default=true]
   [--package-list PACKAGES]
```

#### Examples

##### Edge Device on Ubuntu in Update/Full mode

```commandline
inbc sota --reboot=false
```

##### Edge Device on Ubuntu in Update/Full mode with package list

```commandline
inbc sota --package-list less,git --reboot=false
```

This will install (or upgrade) the less and git packages and any necessary
dependencies.

##### Edge Device on Ubuntu in download-only mode

```commandline
inbc sota --mode download-only --reboot=false
```

##### Edge Device on Ubuntu in download-only mode with package list

```commandline
inbc sota --mode download-only --package-list=less,git --reboot=false
```

This will download the latest versions of less and git and any necessary dependencies.

##### Edge Device on Ubuntu in no-download mode

```commandline
inbc sota --mode no-download --reboot=false
```

##### Edge Device on Ubuntu in no-download mode with package list

```commandline
inbc sota --mode no-download --package-list less,git
```

This will upgrade or install the packages and get any necessary
dependencies, as long as all packages needed to do this have already been
downloaded. (see download-only mode)

### SOURCE APPLICATION ADD

#### Description

Optionally Downloads and encrypts GPG key and stores it on the system under <em>/usr/share/keyrings</em>.  Creates a file under <em>/etc/apt/sources.list.d</em> to store the update source information.
This list file is used during 'sudo apt update' to update the application.  <em>Deb882</em> format may be used instead of downloading a GPG key.

**NOTE:** Make sure to add gpgKeyUri to the trustedrepositories before using INBC source application ADD command

#### Usage

```commandline
inbc source application add
   {--sources SOURCES}
   {--filename FILENAME}
   [--gpgKeyUri GPG_KEY_URI]
   [--gpgKeyName GPG_KEY_NAME]
```

#### Example

##### Add an Application Source (non-deb822 format with remote GPG key)

```commandline
inbc source application add 
   --gpgKeyUri https://dl-ssl.google.com/linux/linux_signing_key.pub 
   --gpgKeyName google-chrome.gpg 
   --sources "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main"  
   --filename google-chrome.list
```

##### Add an Application Source (using deb822 format)

**NOTE:** In the Signed-By: Section, use the following guidelines.

- Each blank line has a period in it. -> " ."
- Each line after the Signed-By: starts with a space -> " gibberish"

```commandline
inbc source application add 
   --sources 
      "Enabled: yes,Types: deb,URIs: http://dl.google.com/linux/chrome/deb/,Suites: stable,Components: main,Signed-By: -----BEGIN PGP PUBLIC KEY BLOCK-----, Version: GnuPG v1.4.2.2 (GNU/Linux), ., mQGiBEXwb0YRBADQva2NLpYXxgjNkbuP0LnPoEXruGmvi3XMIxjEUFuGNCP4Rj/a, kv2E5VixBP1vcQFDRJ+p1puh8NU0XERlhpyZrVMzzS/RdWdyXf7E5S8oqNXsoD1z, -----END PGP PUBLIC KEY BLOCK-----"
   --filename google-chrome.sources
```

### SOURCE APPLICATION REMOVE

#### Description

Removes the source file from under /etc/apt/sources.list.d/.  Optionally removes the GPG key file from under <em>/usr/share/keyrings</em>.

#### Usage

```commandline
inbc source application remove    
   {--filename FILE_NAME}
   [--gpgKeyName GPG_KEY_NAME]
```

#### Example

##### Remove an application source (Both GPG key and Source File)

```commandline
inbc source application remove 
    --gpgKeyName google-chrome.gpg 
    --filename google-chrome.list
```

##### Remove an application source (deb822 format)

```commandline
inbc source application remove 
    --filename google-chrome.sources
```

### SOURCE OS UPDATE

#### Description

Creates a new <em>/etc/apt/sources.list</em> file with only the sources provided

#### Usage

```commandline
inbc source os update
    {--sources SOURCES}
```

#### Example

##### Creates a new <em>/etc/apt/sources.list</em> file with only the two provided sources

- NOTE: list must be comma separated

```commandline
inbc source os update
    --sources "deb http://archive.ubuntu.com/ubuntu/ jammy-security main restricted, deb http://archive.ubuntu.com/ubuntu/ jammy-security universe"
```

## LOAD

### Description

Load a new configuration file.   This will replace the existing configuration file with the new file.

📝 The configuration file you provide needs to be named *intel_manageability.conf*.

### Usage

```commandline
inbc load
   [--uri, -u URI]
   {--signature, -s SIGNATURE}
```

### Examples

#### Load new Configuration File

```commandline
inbc load --uri  <URI to config file>/config.file
```

#### Load new Configuration File with signature

You can load a configuration file with a signature in two ways:

1. **Direct config file signature:**
   The signature must be generated over the config file itself.

   ```commandline
   sudo inbc load --uri <URI to config file>/config.conf --signature "<hex_signature_string>"
   ```

2. **Tarball with PEM and config:**
   The configuration file you provide must be named `intel_manageability.conf`.
   To use a signature with a PEM certificate, create a tar archive containing both the `intel_manageability.conf` file and the PEM certificate file.
   The signature must be generated over the entire tarball, not just the config file.

   ```commandline
   sudo inbc load --uri <URI to config file>/config.tar --signature "<hex_signature_string>"
   ```

**Note:**
- For tarball mode, both `intel_manageability.conf` and the PEM file must be present in the tar archive.
- The signature must match the file or tarball as described above.

## GET

### Description

Get key/value pairs from configuration file

### Usage

```commandline
inbc get
   {--path, -p KEY_PATH;...} 
```

### Examples

#### Get Configuration Value

```commandline
inbc get --path  os_updater.proceedWithoutRollback
```

## SET

### Description

Set key/value pairs in configuration file

### Usage

```commandline
inbc set
   {--path, -p KEY_PATH;...} 
```

### Examples

#### Set Configuration Value

```commandline
inbc set --path  os_updater.proceedWithoutRollback:true
```

## Append

### Description

Append is only applicable to config tags, which is trustedRepositories

### Usage

```commandline
inbc append
   {--path, -p KEY_PATH;...} 
```

### Examples

#### Append a key/value pair

```commandline
inbc append --path  os_updater.trustedRepositories:https://abc.com/
```

## Remove

### Description

Remove is only applicable to config tags, which is trustedRepositories

### Usage

```commandline
inbc remove 
   {--path, -p KEY_PATH;...} 
```

### Examples

#### Remove a key/value pair

```commandline
inbc remove --path  os_updater.trustedRepositories:<https://abc.com/>
```

## QUERY

### Description

Query device(s) for attributes

### Usage

```commandline
inbc query
   [--option, -o=[all | hw | fw |  os | swbom | version ]; default='all']
```

### Examples

#### Return all attributes

```commandline
inbc query
```

#### Return only 'hw' attributes

```commandline
inbc query --option hw
```

#### Return only 'swbom' attributes

```commandline
inbc query --option swbom
```

### Option Results

# Query Command

## Description

The Query command can be called by either the cloud or INBC.  It will provide attribute information on the Host.

## Options

#### 'hw' - Hardware

| Attribute     | Description                                         |
|:--------------|:----------------------------------------------------|
| manufacturer  | Hardware manufacturer                               |
| product       | Product type                                        |
| stepping      | Stepping                                            |
| sku           | SKU                                                 |
| model         | Model number                                        |
| serial_sum    | Serial number                                       |

#### 'fw' - Firmware

| Attribute       | Description      |
|:----------------|:-----------------|
| boot_fw_date    | Firmware date    |
| boot_fw_vendor  | Firmware vendor  |
| boot_fw_version | Firmware version |

#### 'os' - Operating System

| Attribute       | Description                   |
|:----------------|:------------------------------|
| os_type         | Operating System type         |
| os_version      | Operating System version      |
| os_release_date | Operating System release date |

#### 'swbom' - Software BOM

SWBOM dynamic telemetry data

#### 'version' - Version

| Attribute | Description    |
|:----------|:---------------|
| version   | Version number |

## RESTART

### Description

Restart

### Usage

```commandline
inbc restart
```

## SHUTDOWN

### Description

Shutdown

### Usage

```commandline
inbc shutdown
```
