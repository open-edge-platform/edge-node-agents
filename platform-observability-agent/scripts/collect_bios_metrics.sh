#!/bin/sh
# SPDX-FileCopyrightText: 2026 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# ---- BIOS ----
BIOS_VENDOR=$(dmidecode -s bios-vendor 2>/dev/null)
BIOS_VERSION=$(dmidecode -s bios-version 2>/dev/null)
BIOS_DATE=$(dmidecode -s bios-release-date 2>/dev/null)

# ---- SYSTEM ----
SYS_VENDOR=$(dmidecode -s system-manufacturer 2>/dev/null)
PRODUCT=$(dmidecode -s system-product-name 2>/dev/null)

# ---- CPU ----
SOCKETS=$(dmidecode -t processor 2>/dev/null | awk -F: '
/Socket Designation/ {n++}
END {print n+0}
')

CORES=$(dmidecode -t processor 2>/dev/null | awk -F: '
/Core Count/ {
  gsub(/ /, "", $2)
  if ($2 ~ /^[0-9]+$/) sum += $2
}
END {print sum+0}
')

THREADS=$(dmidecode -t processor 2>/dev/null | awk -F: '
/Thread Count/ {
  gsub(/ /, "", $2)
  if ($2 ~ /^[0-9]+$/) sum += $2
}
END {print sum+0}
')

# ---- MEMORY ----
set -- $(
  dmidecode -t memory 2>/dev/null | awk '
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

TOTAL_MEM_MB=${1:-0}
DIMMS=${2:-0}
DIMMS_POPULATED=${3:-0}

# ---- OUTPUT ----
# Escape double quotes for influx string fields
escape_influx_string() {
  echo "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

BIOS_VENDOR_ESC=$(escape_influx_string "$BIOS_VENDOR")
BIOS_VERSION_ESC=$(escape_influx_string "$BIOS_VERSION")
BIOS_DATE_ESC=$(escape_influx_string "$BIOS_DATE")
SYS_VENDOR_ESC=$(escape_influx_string "$SYS_VENDOR")
PRODUCT_ESC=$(escape_influx_string "$PRODUCT")

echo "bios_info \
bios_vendor=\"${BIOS_VENDOR_ESC}\",\
bios_version=\"${BIOS_VERSION_ESC}\",\
bios_date=\"${BIOS_DATE_ESC}\",\
system_vendor=\"${SYS_VENDOR_ESC}\",\
product=\"${PRODUCT_ESC}\",\
cpu_sockets=${SOCKETS}i,\
cpu_cores=${CORES}i,\
cpu_threads=${THREADS}i,\
mem_total_mb=${TOTAL_MEM_MB}i,\
mem_dimms=${DIMMS}i,\
mem_dimms_populated=${DIMMS_POPULATED}i"
