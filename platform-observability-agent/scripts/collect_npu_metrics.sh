#!/bin/sh
# SPDX-FileCopyrightText: 2026 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

#!/bin/sh

CAT=/usr/bin/cat
BASENAME=/usr/bin/basename
READLINK=/usr/bin/readlink
JQ=/usr/bin/jq

# Validate required tooling
for cmd in "$CAT" "$BASENAME" "$READLINK" "$JQ"; do
  if [ ! -x "$cmd" ]; then
    echo '{"accel":"none","pci":"none","driver":"none","present":0,"collection_status":0}'
    exit 0
  fi
done

numeric_or_zero() {
  case "$1" in
    ''|*[^0-9]* ) echo 0 ;;
    *) echo "$1" ;;
  esac
}

found=0

for acc in /sys/class/accel/accel*; do
  [ -e "$acc" ] || continue
  found=1

  ACCEL=$("$BASENAME" "$acc")
  DEV_PATH=$("$READLINK" -f "$acc/device")
  PCI_BDF=$("$BASENAME" "$DEV_PATH")

  DRIVER="unknown"
  if [ -L "$DEV_PATH/driver" ]; then
    DRIVER=$("$BASENAME" "$("$READLINK" -f "$DEV_PATH/driver")")
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
    RUNTIME_STATUS=$("$CAT" "$POWER_PATH/runtime_status" 2>/dev/null || echo unknown)
    [ "$RUNTIME_STATUS" = "active" ] && RUNTIME_ACTIVE=1
  fi

  if [ -f "$POWER_PATH/runtime_active_time" ]; then
    RUNTIME_ACTIVE_TIME=$("$CAT" "$POWER_PATH/runtime_active_time" 2>/dev/null || echo 0)
  fi

  if [ -f "$POWER_PATH/runtime_suspended_time" ]; then
    RUNTIME_SUSPENDED_TIME=$("$CAT" "$POWER_PATH/runtime_suspended_time" 2>/dev/null || echo 0)
  fi

  if [ -f "$POWER_PATH/runtime_usage" ]; then
    RUNTIME_USAGE=$("$CAT" "$POWER_PATH/runtime_usage" 2>/dev/null || echo 0)
  fi

  RUNTIME_ACTIVE_TIME=$(numeric_or_zero "$RUNTIME_ACTIVE_TIME")
  RUNTIME_SUSPENDED_TIME=$(numeric_or_zero "$RUNTIME_SUSPENDED_TIME")
  RUNTIME_USAGE=$(numeric_or_zero "$RUNTIME_USAGE")

  RUNTIME_TOTAL_TIME=$((RUNTIME_ACTIVE_TIME + RUNTIME_SUSPENDED_TIME))

  "$JQ" -n \
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
    --argjson collection_status 1 \
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
      runtime_usage: $runtime_usage,
      collection_status: $collection_status
    }'

  exit 0
done

if [ "$found" -eq 0 ]; then
  "$JQ" -n \
    '{
      accel: "none",
      pci: "none",
      driver: "none",
      present: 0,
      collection_status: 0
    }'
fi
