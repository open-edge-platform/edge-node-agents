#!/bin/sh
# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Get metrics for all disks in lsblk
DISKTOTALS="[]"
DISKPARTUSED="[]"
DISKLVMUSED="[]"
DISKAVAIL="[]"

LSBLKDISKINFO=$(test -h /usr/bin/lsblk || /usr/bin/lsblk -o NAME,TYPE -l)
DISKNAME=""

for disk in ${LSBLKDISKINFO}; do
	if [ "${disk}" = "NAME" ] || [ "${disk}" = "TYPE" ]; then
		continue
	fi
	if [ "${DISKNAME}" = "" ]; then
		DISKNAME=${disk}
	else
		if [ "${disk}" = "disk" ]; then
			DISKSIZE=$(test -h /usr/bin/lsblk || test -h /usr/bin/grep || test -h /usr/bin/cut || /usr/bin/lsblk -o SIZE,NAME -b -l /dev/"${DISKNAME}" | /usr/bin/grep -w "${DISKNAME}" | /usr/bin/cut -d ' ' -f 1)
			if [ "${DISKSIZE}" = "" ]; then
				DISKSIZE=0
			fi
			DISKPARTITIONS=$(test -h /usr/bin/lsblk || /usr/bin/lsblk -o NAME -b -l /dev/"${DISKNAME}")
			TOTALPARTUSED=0
			TOTALLVMASSIGNED=0
			CHECKIFDISK=0
			for partition in ${DISKPARTITIONS}; do
				if [ "${partition}" = "${DISKNAME}" ] || [ ${CHECKIFDISK} -eq 0 ]; then
					CHECKIFDISK=1
					continue
				fi
				PARTTYPE=$(test -h /usr/bin/lsblk || test -h /usr/bin/grep || test -h /usr/bin/cut || /usr/bin/lsblk -o TYPE,NAME -l | /usr/bin/grep -w "${partition}" | /usr/bin/cut -d ' ' -f 1)
				if [ "${PARTTYPE}" = "" ]; then
					continue
				fi
				if [ "${PARTTYPE}" = "lvm" ]; then
					LVMSIZE=$(test -h /usr/bin/lsblk || test -h /usr/bin/grep || test -h /usr/bin/cut || /usr/bin/lsblk -o NAME,SIZE -l -b | /usr/bin/grep -w "${partition}" | /usr/bin/cut -d ' ' -f2-)
					if [ "${LVMSIZE}" = "" ]; then
						LVMSIZE=0
					fi
					TOTALLVMASSIGNED=$((TOTALLVMASSIGNED+LVMSIZE))
				else
					if [ "${partition}" = "crypt" ] && [ "${partition}" != "rootfs_crypt" ]; then
						continue
					fi
					PARTSIZE=$(test -h /usr/bin/lsblk || test -h /usr/bin/grep || test -h /usr/bin/cut || /usr/bin/lsblk -o NAME,SIZE -b -l | /usr/bin/grep -w "${partition}" | /usr/bin/cut -d ' ' -f2-)
					if [ "${PARTSIZE}" = "" ]; then
						PARTSIZE=0
					fi
					PARTUSEDPERCENT=$(test -h /usr/bin/df || test -h /usr/bin/grep || test -h /usr/bin/cut || /usr/bin/df --output=source,pcent | /usr/bin/grep -w "${partition}" | /usr/bin/cut -d '%' -f 1 | /usr/bin/cut -d ' ' -f 2-)
					if [ "${PARTUSEDPERCENT}" = "" ]; then
						PARTUSEDPERCENT=0
					fi
					PARTSIZEUSED=$(($((PARTSIZE*PARTUSEDPERCENT))/100))
					TOTALPARTUSED=$((TOTALPARTUSED+PARTSIZEUSED))
				fi
			done
			TOTALAVAIL=$((DISKSIZE-$((TOTALPARTUSED+TOTALLVMASSIGNED))))
			DISKTOTALS=$(test -h /usr/bin/echo || test -h /usr/bin/jq || /usr/bin/echo "$DISKTOTALS" | /usr/bin/jq --arg disk_size_total_bytes "${DISKSIZE}" --arg tag "${DISKNAME}" '. += [{$disk_size_total_bytes, $tag}]')
			DISKPARTUSED=$(test -h /usr/bin/echo || test -h /usr/bin/jq || /usr/bin/echo "$DISKPARTUSED" | /usr/bin/jq --arg disk_size_used_partition_bytes "${TOTALPARTUSED}" --arg tag "${DISKNAME}" '. += [{$disk_size_used_partition_bytes, $tag}]')
			DISKLVMUSED=$(test -h /usr/bin/echo || test -h /usr/bin/jq || /usr/bin/echo "$DISKLVMUSED" | /usr/bin/jq --arg disk_size_used_lvm_bytes "${TOTALLVMASSIGNED}" --arg tag "${DISKNAME}" '. += [{$disk_size_used_lvm_bytes, $tag}]')
			DISKAVAIL=$(test -h /usr/bin/echo || test -h /usr/bin/jq || /usr/bin/echo "$DISKAVAIL" | /usr/bin/jq --arg disk_size_available_bytes "${TOTALAVAIL}" --arg tag "${DISKNAME}" '. += [{$disk_size_available_bytes, $tag}]')
		fi
		DISKNAME=""
	fi
done

if [ "${DISKTOTALS}" = "" ]; then
	DISKTOTALS="[]"
fi
if [ "${DISKPARTUSED}" = "" ]; then
	DISKPARTUSED="[]"
fi
if [ "${DISKLVMUSED}" = "" ]; then
	DISKLVMUSED="[]"
fi
if [ "${DISKAVAIL}" = "" ]; then
	DISKAVAIL="[]"
fi

JSON="{}"
JSON=$(test -h /usr/bin/echo || test -h /usr/bin/jq || /usr/bin/echo $JSON | /usr/bin/jq --argjson diskSizeTotal "${DISKTOTALS}" --argjson diskPartUsedTotal "${DISKPARTUSED}" --argjson diskLvmUsedTotal "${DISKLVMUSED}" --argjson diskAvailTotal "${DISKAVAIL}" '. += {$diskSizeTotal, $diskPartUsedTotal, $diskLvmUsedTotal, $diskAvailTotal}')
if [ "${JSON}" != "" ]; then
	test -h /usr/bin/echo || /usr/bin/echo "$JSON"
fi
