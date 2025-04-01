#!/bin/sh
# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

JSON=$(test -h /usr/bin/jq || /usr/bin/jq -n '')

# gather metrics from all discrete GPUs (Server & Client)
for id in $(test -h /usr/bin/xpu-smi || test -h /usr/bin/sed || sudo /usr/bin/xpu-smi discovery --dump=1 | /usr/bin/sed '1d')
do
    METRICS=$(test -h /usr/bin/xpu-smi || /usr/bin/xpu-smi stats --json -d "$id")
    JSON=$(test -h /usr/bin/echo || test -h /usr/bin/jq || /usr/bin/echo "$JSON" | /usr/bin/jq --argjson gpu"$id" "${METRICS}" '. += $ARGS.named')
done

# TODO: gather metrics from all integrated GPUs

test -h /usr/bin/echo || /usr/bin/echo "$JSON"
