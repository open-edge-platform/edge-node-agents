#!/bin/sh
# SPDX-FileCopyrightText: 2026 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

#!/bin/sh

DMIDECODE=/usr/sbin/dmidecode
AWK=/usr/bin/awk
JQ=/usr/bin/jq

# Validate required binaries
for cmd in "$DMIDECODE" "$JQ"; do
  if [ ! -x "$cmd" ]; then
    echo '{"collection_status":0}'
    exit 0
  fi
done

numeric_or_zero() {
  case "$1" in
    ''|*[^0-9]* ) echo 0 ;;
    *) echo "$1" ;;
  esac
}

# ---- BIOS ----
BIOS_VENDOR=$("$DMIDECODE" -s bios-vendor 2>/dev/null)
BIOS_VERSION=$("$DMIDECODE" -s bios-version 2>/dev/null)
BIOS_DATE=$("$DMIDECODE" -s bios-release-date 2>/dev/null)

# ---- SYSTEM ----
SYS_VENDOR=$("$DMIDECODE" -s system-manufacturer 2>/dev/null)
PRODUCT=$("$DMIDECODE" -s system-product-name 2>/dev/null)

# ---- CPU ----
SOCKETS=$("$DMIDECODE" -t processor 2>/dev/null | "$AWK" -F: '
/Socket Designation/ {n++}
END {print n+0}
')

CORES=$("$DMIDECODE" -t processor 2>/dev/null | "$AWK" -F: '
/Core Count/ {
  gsub(/ /, "", $2)
  if ($2 ~ /^[0-9]+$/) sum += $2
}
END {print sum+0}
')

THREADS=$("$DMIDECODE" -t processor 2>/dev/null | "$AWK" -F: '
/Thread Count/ {
  gsub(/ /, "", $2)
  if ($2 ~ /^[0-9]+$/) sum += $2
}
END {print sum+0}
')

# ---- MEMORY ----
set -- $(
  "$DMIDECODE" -t memory 2>/dev/null | "$AWK" '
    BEGIN {
      total=0
      dimms=0
      populated=0
    }

    /^[[:space:]]*Size:/ {
      dimms++
      if ($2 ~ /^[0-9]+$/ && $3 == "MB") {
        total += $2
        populated++
      } else if ($2 ~ /^[0-9]+$/ && $3 == "GB") {
        total += ($2 * 1024)
        populated++
      }
    }

    END {
      print total, dimms, populated
    }
  '
)

TOTAL_MEM_MB=$(numeric_or_zero "${1:-0}")
DIMMS=$(numeric_or_zero "${2:-0}")
DIMMS_POPULATED=$(numeric_or_zero "${3:-0}")
SOCKETS=$(numeric_or_zero "$SOCKETS")
CORES=$(numeric_or_zero "$CORES")
THREADS=$(numeric_or_zero "$THREADS")

# ---- OUTPUT JSON ----
"$JQ" -n \
  --arg bios_vendor "$BIOS_VENDOR" \
  --arg bios_version "$BIOS_VERSION" \
  --arg bios_date "$BIOS_DATE" \
  --arg system_vendor "$SYS_VENDOR" \
  --arg product "$PRODUCT" \
  --argjson cpu_sockets "$SOCKETS" \
  --argjson cpu_cores "$CORES" \
  --argjson cpu_threads "$THREADS" \
  --argjson mem_total_mb "$TOTAL_MEM_MB" \
  --argjson mem_dimms "$DIMMS" \
  --argjson mem_dimms_populated "$DIMMS_POPULATED" \
  --argjson collection_status 1 \
  '{
    bios_vendor: $bios_vendor,
    bios_version: $bios_version,
    bios_date: $bios_date,
    system_vendor: $system_vendor,
    product: $product,
    cpu_sockets: $cpu_sockets,
    cpu_cores: $cpu_cores,
    cpu_threads: $cpu_threads,
    mem_total_mb: $mem_total_mb,
    mem_dimms: $mem_dimms,
    mem_dimms_populated: $mem_dimms_populated,
    collection_status: $collection_status
  }'
