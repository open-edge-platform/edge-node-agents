#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# List of npm packages to install
PIPX_PACKAGES=(
  "reuse==5.0.2"
)

# Check if the .tool-versions file path is provided
if [ -z "$1" ]; then
  echo "Error: Path to .tool-versions file is required" >&2
  exit 1
fi

TOOL_VERSIONS_FILE="$1"

# Function to install a tool using asdf
install_tool() {
  local tool=$1
  local version=$2

  # Check if the plugin is installed, if not, install it
  if ! asdf plugin-list | grep -q "^$tool\$"; then
    echo "Installing plugin for $tool..."
    if ! asdf plugin-add "$tool"; then
      echo "Error: Failed to install plugin for $tool" >&2
      return
    else
      echo "Successfully installed plugin for $tool"
    fi
  fi

  # Check if the tool version is installed, if not, install it
  if asdf list "$tool" | grep -q "$version"; then
    echo "$tool $version is already installed."
  else
    echo "Installing $tool $version..."
    if ! asdf install "$tool" "$version"; then
      echo "Error: Failed to install $tool $version" >&2
      return
    else
      echo "Successfully installed $tool $version"
    fi
  fi
}

# Function to install pipx packages
install_pipx_packages() {
  for package in "${PIPX_PACKAGES[@]}"; do
    if pipx install "$package"; then
      echo "Successfully installed $package"
    else
      echo "Error installing $package" >&2
    fi
  done
}

# Read the .tool-versions file and install each tool
while read -r line; do
  # Skip empty lines
  if [[ -z "$line" ]]; then
    continue
  fi

  tool=$(echo "$line" | awk '{print $1}')
  version=$(echo "$line" | awk '{print $2}')
  install_tool "$tool" "$version"
done < "$TOOL_VERSIONS_FILE"

# Install go packages
go install github.com/boumenot/gocover-cobertura@v1.3.0

# Call the function to install pipx packages
install_pipx_packages

# Ensure the installation path is in $PATH
pipx ensurepath
