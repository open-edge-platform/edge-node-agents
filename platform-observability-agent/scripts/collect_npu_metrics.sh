#!/bin/sh
# SPDX-FileCopyrightText: 2026 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Escape special characters in influx line protocol tag values
escape_influx_tag() {
  echo "$1" | sed 's/\\/\\\\/g; s/,/\\,/g; s/ /\\ /g; s/=/\\=/g'
}

# Escape double quotes for influx string fields
escape_influx_string() {
  echo "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

found=0

for acc in /sys/class/accel/accel*; do
  [ -e "$acc" ] || continue
  found=1

  ACCEL=$(basename "$acc")
  DEV_PATH=$(readlink -f "$acc/device")
  PCI_BDF=$(basename "$DEV_PATH")

  DRIVER="unknown"
  if [ -L "$DEV_PATH/driver" ]; then
    DRIVER=$(basename "$(readlink -f "$DEV_PATH/driver")")
  fi

  PRESENT=0
  [ -e "/dev/$ACCEL" ] && PRESENT=1
  [ -e "/dev/accel/$ACCEL" ] && PRESENT=1

  DRIVER_BOUND=0
  [ "$DRIVER" != "unknown" ] && DRIVER_BOUND=1

  RUNTIME_STATUS="unknown"
  RUNTIME_ACTIVE=0
  RUNTIME_ACTIVE_TIME=0
  RUNTIME_SUSPENDED_TIME=0
  RUNTIME_USAGE=0

  POWER_PATH="$acc/power"

  if [ -f "$POWER_PATH/runtime_status" ]; then
    RUNTIME_STATUS=$(cat "$POWER_PATH/runtime_status" 2>/dev/null || echo unknown)
    [ "$RUNTIME_STATUS" = "active" ] && RUNTIME_ACTIVE=1
  fi

  if [ -f "$POWER_PATH/runtime_active_time" ]; then
    RUNTIME_ACTIVE_TIME=$(cat "$POWER_PATH/runtime_active_time" 2>/dev/null || echo 0)
  fi

  if [ -f "$POWER_PATH/runtime_suspended_time" ]; then
    RUNTIME_SUSPENDED_TIME=$(cat "$POWER_PATH/runtime_suspended_time" 2>/dev/null || echo 0)
  fi

  if [ -f "$POWER_PATH/runtime_usage" ]; then
    RUNTIME_USAGE=$(cat "$POWER_PATH/runtime_usage" 2>/dev/null || echo 0)
  fi

  RUNTIME_TOTAL_TIME=$((RUNTIME_ACTIVE_TIME + RUNTIME_SUSPENDED_TIME))

  ACCEL_TAG=$(escape_influx_tag "$ACCEL")
  PCI_TAG=$(escape_influx_tag "$PCI_BDF")
  DRIVER_TAG=$(escape_influx_tag "$DRIVER")
  STATUS_ESC=$(escape_influx_string "$RUNTIME_STATUS")

  echo "npu_metrics,accel=${ACCEL_TAG},pci=${PCI_TAG},driver=${DRIVER_TAG} \
runtime_status=\"${STATUS_ESC}\",\
present=${PRESENT}i,\
driver_bound=${DRIVER_BOUND}i,\
runtime_active=${RUNTIME_ACTIVE}i,\
runtime_active_time_ms=${RUNTIME_ACTIVE_TIME}i,\
runtime_suspended_time_ms=${RUNTIME_SUSPENDED_TIME}i,\
runtime_total_time_ms=${RUNTIME_TOTAL_TIME}i,\
runtime_usage=${RUNTIME_USAGE}i"
done

if [ "$found" -eq 0 ]; then
  echo "npu_metrics,accel=none,pci=none,driver=none present=0i,collection_status=0i"
fi
