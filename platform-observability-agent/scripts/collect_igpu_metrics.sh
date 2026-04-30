#!/bin/sh
# SPDX-FileCopyrightText: 2026 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# ---- Full paths ----
TIMEOUT=/usr/bin/timeout
STDBUF=/usr/bin/stdbuf
INTEL_GPU_TOP=/usr/bin/intel_gpu_top
JQ=/usr/bin/jq
AWK=/usr/bin/awk
CAT=/usr/bin/cat
BASENAME=/usr/bin/basename
READLINK=/usr/bin/readlink
GREP=/usr/bin/grep
HEAD=/usr/bin/head

# ---- Validate critical tools ----
for cmd in "$TIMEOUT" "$STDBUF" "$INTEL_GPU_TOP" "$JQ"; do
  if [ ! -x "$cmd" ]; then
    echo '{"collection_status":0}'
    exit 0
  fi
done

numeric_or_zero() {
  case "$1" in
    ''|*[^0-9.]* ) echo 0 ;;
    *) echo "$1" ;;
  esac
}

found=0

for d in /sys/class/drm/card*; do
  [ -e "$d/device/vendor" ] || continue

  VENDOR=$($CAT "$d/device/vendor")
  PCI_ADDR=$($BASENAME "$($READLINK -f "$d/device")")

  # Intel iGPU filter
  case "$PCI_ADDR" in
    *0000:00:02.0*) IS_IGPU=1 ;;
    *) IS_IGPU=0 ;;
  esac

  if [ "$VENDOR" != "0x8086" ] || [ "$IS_IGPU" -ne 1 ]; then
    continue
  fi

  found=1

  CARD=$($BASENAME "$d")
  CARD_NUM=${CARD#card}
  DRI_PATH="/sys/kernel/debug/dri/$CARD_NUM"

  JSON=$("$TIMEOUT" 3 "$STDBUF" -oL "$INTEL_GPU_TOP" -J -d "drm:/dev/dri/$CARD" -s 1000 2>/dev/null | "$AWK" '
    /^{/ {buf=$0; depth=1; next}
    depth>0 {
      buf = buf "\n" $0
      depth += gsub(/{/, "{") - gsub(/}/, "}")
      if (depth == 0 && buf ~ /"engines"/) {
        print buf
        exit
      }
    }')

  if [ -z "$JSON" ]; then
    "$JQ" -n --arg card "$CARD" \
      '{card:$card, collection_status:0}'
    exit 0
  fi

  BUSY=$(echo "$JSON" | "$JQ" -r '[.engines[]?.busy // 0] | add // 0' 2>/dev/null)
  RC6=$(echo "$JSON" | "$JQ" -r '.rc6.value // 0' 2>/dev/null)
  FREQ=$(echo "$JSON" | "$JQ" -r '.frequency.actual // 0' 2>/dev/null)
  POWER=$(echo "$JSON" | "$JQ" -r '.power.GPU // 0' 2>/dev/null)

  BUSY=$(numeric_or_zero "$BUSY")
  RC6=$(numeric_or_zero "$RC6")
  FREQ=$(numeric_or_zero "$FREQ")
  POWER=$(numeric_or_zero "$POWER")

  MEM_BYTES=0
  if [ -f "$DRI_PATH/i915_gem_objects" ]; then
    MEM_BYTES=$($GREP -o '[0-9]\+ bytes' "$DRI_PATH/i915_gem_objects" | "$AWK" '{print $1}' | "$HEAD" -n 1)
    MEM_BYTES=${MEM_BYTES:-0}
  fi
  MEM_BYTES=$(numeric_or_zero "$MEM_BYTES")

  "$JQ" -n \
    --arg card "$CARD" \
    --argjson collection_status 1 \
    --argjson engine_busy_pct "$BUSY" \
    --argjson rc6_residency_pct "$RC6" \
    --argjson freq_mhz "$FREQ" \
    --argjson power_w "$POWER" \
    --argjson mem_bytes "$MEM_BYTES" \
    '{
      card: $card,
      collection_status: $collection_status,
      engine_busy_pct: $engine_busy_pct,
      rc6_residency_pct: $rc6_residency_pct,
      freq_mhz: $freq_mhz,
      power_w: $power_w,
      mem_bytes: $mem_bytes
    }'

  exit 0
done

if [ "$found" -eq 0 ]; then
  echo '{"collection_status":0}'
fi
