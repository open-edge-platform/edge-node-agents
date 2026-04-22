#!/bin/sh
# SPDX-FileCopyrightText: 2026 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Loop over DRM cards
for d in /sys/class/drm/card*; do
  [ -e "$d/device/vendor" ] || continue

  VENDOR=$(cat "$d/device/vendor")
  PCI_ADDR=$(basename "$(readlink -f $d/device)")

  # ---- Filter ONLY Intel iGPU (00:02.0) ----
  if [ "$VENDOR" != "0x8086" ] || [[ "$PCI_ADDR" != *"0000:00:02.0"* ]]; then
    continue
  fi

  CARD=$(basename "$d")
  CARD_NUM=${CARD#card}
  DRI_PATH="/sys/kernel/debug/dri/$CARD_NUM"

  # ---- Capture JSON ----
  JSON=$(timeout 3 stdbuf -oL intel_gpu_top -J -d drm:/dev/dri/$CARD -s 1000 2>/dev/null | awk '
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
    echo "igpu_metrics,card=$CARD collection_status=0i"
    continue
  fi

  # ---- Parse metrics ----
  BUSY=$(echo "$JSON" | jq -r '.engines | to_entries[0].value.busy // 0' 2>/dev/null)
  RC6=$(echo "$JSON" | jq -r '.rc6.value // 0' 2>/dev/null)
  FREQ=$(echo "$JSON" | jq -r '.frequency.actual // 0' 2>/dev/null)
  POWER=$(echo "$JSON" | jq -r '.power.GPU // 0' 2>/dev/null)

  BUSY=${BUSY:-0}
  RC6=${RC6:-0}
  FREQ=${FREQ:-0}
  POWER=${POWER:-0}

  # ---- Memory (UMA) ----
  MEM_BYTES=0
  if [ -f "$DRI_PATH/i915_gem_objects" ]; then
    MEM_BYTES=$(grep -o '[0-9]\+ bytes' "$DRI_PATH/i915_gem_objects" | awk '{print $1}')
    MEM_BYTES=${MEM_BYTES:-0}
  fi

  # ---- Output (InfluxDB line protocol) ----
  echo "igpu_metrics,card=${CARD} engine_busy_pct=${BUSY},rc6_residency_pct=${RC6},freq_mhz=${FREQ},power_w=${POWER},mem_bytes=${MEM_BYTES}i" | tr -d '\000' | head -n 1

done
