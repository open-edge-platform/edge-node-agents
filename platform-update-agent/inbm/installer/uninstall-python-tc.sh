#!/bin/bash
set -eo pipefail

# Uninstaller for Python-based Intel In-Band Manageability framework
# This script removes all Python-based INBM components

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

echo "=== Uninstalling Python-based Intel In-Band Manageability framework ==="

# Stop and disable all Python-based services
echo "Stopping and disabling all Python-based services..."
for service in inbm-configuration inbm-dispatcher inbm-telemetry inbm-diagnostic inbm-cloudadapter mqtt inbm configuration dispatcher telemetry diagnostic cloudadapter; do
    systemctl disable --now "${service}" >/dev/null 2>&1 || true
done

# Remove all Python-based packages
echo "Removing all Python-based packages..."
dpkg --remove --force-all \
    inbm-configuration-agent configuration-agent \
    inbm-dispatcher-agent dispatcher-agent \
    inbm-diagnostic-agent diagnostic-agent \
    inbm-cloudadapter-agent cloudadapter-agent \
    inbm-telemetry-agent telemetry-agent \
    mqtt-agent trtl mqtt \
    no-tpm-provision tpm-provision \
    inbm-configuration inbm-dispatcher inbm-telemetry inbm-diagnostic inbm-cloudadapter \
    >/dev/null 2>&1 || true

# Clean up configuration directories and files
echo "Cleaning up configuration files and directories..."
rm -rf /etc/intel-manageability/dispatcher* >/dev/null 2>&1 || true
rm -rf /etc/intel-manageability/telemetry* >/dev/null 2>&1 || true
rm -rf /etc/intel-manageability/configuration* >/dev/null 2>&1 || true
rm -rf /etc/intel-manageability/diagnostic* >/dev/null 2>&1 || true
rm -rf /etc/intel-manageability/cloudadapter* >/dev/null 2>&1 || true
rm -f /etc/intel-manageability/mqtt* >/dev/null 2>&1 || true
rm -f /etc/intel-manageability/*.xml >/dev/null 2>&1 || true

# Remove systemd service files
echo "Removing systemd service files..."
rm -f /lib/systemd/system/inbm-*.service >/dev/null 2>&1 || true
rm -f /etc/systemd/system/inbm-*.service >/dev/null 2>&1 || true
rm -f /lib/systemd/system/mqtt.service >/dev/null 2>&1 || true
systemctl daemon-reload

# Remove executables and scripts
echo "Removing executables and scripts..."
rm -f /usr/bin/inbm-* >/dev/null 2>&1 || true
rm -rf /usr/share/inbm* >/dev/null 2>&1 || true
rm -rf /usr/lib/python*/site-packages/inbm* >/dev/null 2>&1 || true

# Clean up data directories
echo "Cleaning up data directories..."
rm -rf /var/cache/manageability/downloads >/dev/null 2>&1 || true
rm -rf /var/log/inbm* >/dev/null 2>&1 || true
rm -rf /var/lib/inbm* >/dev/null 2>&1 || true

# Clean up user accounts (be careful here)
for user in mqtt-broker dispatcher-agent telemetry-agent configuration-agent diagnostic-agent cloudadapter-agent; do
    if id "$user" >/dev/null 2>&1; then
        echo "Removing user account: $user"
        userdel "$user" >/dev/null 2>&1 || true
    fi
done

# Clean up groups
for group in mqtt-broker dispatcher-agent telemetry-agent configuration-agent diagnostic-agent cloudadapter-agent; do
    if getent group "$group" >/dev/null 2>&1; then
        echo "Removing group: $group"
        groupdel "$group" >/dev/null 2>&1 || true
    fi
done

# Remove Python dependencies that were specifically installed for INBM
echo "Cleaning up Python packages..."
pip3 uninstall -y paho-mqtt xmlschema defusedxml jsonschema requests cryptography >/dev/null 2>&1 || true

# Clean up AppArmor profiles
echo "Cleaning up AppArmor profiles..."
rm -f /etc/apparmor.d/usr.bin.inbm-* >/dev/null 2>&1 || true
apparmor_parser -R /etc/apparmor.d/usr.bin.inbm-* >/dev/null 2>&1 || true

# Remove cron jobs
echo "Cleaning up cron jobs..."
rm -f /etc/cron.d/inbm* >/dev/null 2>&1 || true

echo "=== Python-based Intel In-Band Manageability framework uninstallation completed ==="
exit 0