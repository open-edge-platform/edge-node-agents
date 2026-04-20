#!/bin/sh
# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

found=0
JSON=$(/usr/bin/jq -n '{}' 2>/dev/null || /usr/bin/echo '{}')

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

  METRICS=$(/usr/bin/jq -n \
    --arg accel "$ACCEL" \
    --arg pci "$PCI_BDF" \
    --arg driver "$DRIVER" \
    --arg runtime_status "$RUNTIME_STATUS" \
    --argjson present "$PRESENT" \
    --argjson driver_bound "$DRIVER_BOUND" \
    --argjson runtime_active "$RUNTIME_ACTIVE" \
    --argjson runtime_active_time_ms "$RUNTIME_ACTIVE_TIME" \
    --argjson runtime_suspended_time_ms "$RUNTIME_SUSPENDED_TIME" \
    --argjson runtime_total_time_ms "$RUNTIME_TOTAL_TIME" \
    --argjson runtime_usage "$RUNTIME_USAGE" \
    '{
      accel: $accel,
      pci: $pci,
      driver: $driver,
      runtime_status: $runtime_status,
      present: $present,
      driver_bound: $driver_bound,
      runtime_active: $runtime_active,
      runtime_active_time_ms: $runtime_active_time_ms,
      runtime_suspended_time_ms: $runtime_suspended_time_ms,
      runtime_total_time_ms: $runtime_total_time_ms,
      runtime_usage: $runtime_usage
    }' 2>/dev/null || /usr/bin/echo '{}')

  JSON=$(/usr/bin/echo "$JSON" | /usr/bin/jq --arg key "$ACCEL" --argjson val "$METRICS" '. + {($key): $val}' 2>/dev/null || /usr/bin/echo "$JSON")
done

if [ "$found" -eq 0 ]; then
  JSON=$(/usr/bin/jq -n '{
    present: 0,
    driver_bound: 0,
    runtime_active: 0,
    runtime_active_time_ms: 0,
    runtime_suspended_time_ms: 0,
    runtime_total_time_ms: 0,
    runtime_usage: 0
  }' 2>/dev/null || /usr/bin/echo '{}')
fi

/usr/bin/echo "$JSON"
