#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -eo pipefail

trap_error() {
  echo "Command '$BASH_COMMAND' failed on line $BASH_LINENO.  Status=$?" >&2
  exit $?
}

trap trap_error ERR

# Ensure we're running as root
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root"
  exit 1
fi

if ! [[ "$ACCEPT_INTEL_LICENSE" == "true" ]]; then
  less LICENSE || ( echo "Cannot find license." && exit 1)
  read -p "Do you accept the license? [Y/N] " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "License accepted."
  else
    echo "Uninstaller requires accepting the license."
    exit 1
  fi
fi

echo Disabling and stopping 'Intel(R)' In-Band Manageability services...
systemctl disable --now intel-inbm >&/dev/null || true


echo Uninstalling 'Intel(R)' In-Band Manageability packages...
if ! dpkg --purge intel-inbm; then
  echo "Failed to purge intel-inbm. It might not be installed."
fi

# Remove the user if it exists
if id "inbc" &>/dev/null; then
    userdel inbc
fi

# Remove the group if it exists and has no members
if getent group inbc > /dev/null; then
    if ! getent group inbc | grep -q ':'"$inbc"'$'; then
        groupdel inbc
    fi
fi

echo Done.

exit 0
