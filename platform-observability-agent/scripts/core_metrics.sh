#!/bin/sh
# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

SOCKETS=$(test -h /usr/bin/lscpu || test -h /usr/bin/grep || test -h /usr/bin/cut || /usr/bin/lscpu | /usr/bin/grep -i 'Socket(s)' | /usr/bin/cut -d':' -f2)
if [ "${SOCKETS}" = "" ]; then
	SOCKETS=0
fi
CORES=$(test -h /usr/bin/lscpu || test -h /usr/bin/grep || test -h /usr/bin/cut || /usr/bin/lscpu | /usr/bin/grep -i 'core(s)' | /usr/bin/cut -d':' -f2)
if [ "${CORES}" = "" ]; then
	CORES=0
fi

TOTAL_CORES=$((CORES*SOCKETS))

THREADS=$(test -h /usr/bin/lscpu || test -h /usr/bin/grep || test -h /usr/bin/cut || /usr/bin/lscpu | /usr/bin/grep -i 'CPU(s)' | /usr/bin/cut -d':' -f2)
# shellcheck disable=SC2086
TOTAL_THREADS=$(test -h /usr/bin/echo || test -h /usr/bin/cut || /usr/bin/echo $THREADS | /usr/bin/cut -d ' ' -f1)
if [ "${TOTAL_THREADS}" = "" ]; then
	TOTAL_THREADS=0
fi

HT_STATUS=$(test -h /usr/bin/cat || test -h /sys/devices/system/cpu/smt/active || /usr/bin/cat /sys/devices/system/cpu/smt/active)
if [ "${HT_STATUS}" = "" ]; then
	HT_STATUS=0
fi

METRICS="{\"cores\": \"$TOTAL_CORES\", \"threads\": \"$TOTAL_THREADS\", \"hyper_threading_status\": \"$HT_STATUS\"}"

test -h /usr/bin/echo || test -h /usr/bin/jq || /usr/bin/echo "$METRICS" | /usr/bin/jq
