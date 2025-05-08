#!/usr/bin/env bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

return_code=0

# Check version in manifest file
manifest_version=$(yq '.metadata.release' < ena-manifest.yaml)
echo "manifest_version: ${manifest_version}"
new_tag="ena-manifest/${manifest_version}"

# Check if non-dev version and that tag was not already created
is_dev_version=$(echo "${manifest_version}" | grep -e '-dev')
echo "Existing git tags"
existing_tags=$(git tag -l)
echo "$existing_tags"

function tag_check {
	if [ "${is_dev_version}" != "" ]; then
		echo "Manifest version is dev version, skipping version check"
	else
		for tag in ${existing_tags}; do
			if [ "${new_tag}" == "${tag}" ]; then
				echo "ERROR: Duplicate tag: ${tag}"
				return_code=2
			fi
		done
	fi
}

function new_version_check {
	# Check that previous release was done for last dev verison if new dev version
	major_ver=$(echo "${manifest_version}" | cut -d '.' -f 1)
	minor_ver=$(echo "${manifest_version}" | cut -d '.' -f 2)
	patch_ver=$(echo "${manifest_version}" | cut -d '.' -f 3)
	if [ "${is_dev_version}" != "" ]; then
		patch_ver=$(echo "${patch_ver}" | cut -d '-' -f 1)
	fi
	local found_prev_tag=false

	if [ "${major_ver}" == 1 ] && [ "${minor_ver}" == 0 ] && [ "${patch_ver}" == 0 ]; then
		echo "Initial version of manifest file, skip check"
		found_prev_tag=true
	elif [ "${patch_ver}" == 0 ]; then
		prev_minor=$(( minor_ver - 1))
		for tag in ${existing_tags}; do
			check_prefix=$(echo "${tag}" | grep 'ena-manifest')
			if [ "${check_prefix}" != "" ]; then
				tag_major_ver=$(echo "${tag}" | cut -d '.' -f 1 | cut -d '/' -f 2)
				tag_minor_ver=$(echo "${tag}" | cut -d '.' -f 2)
				if [ "${tag_major_ver}" == "${major_ver}" ] && [ "${tag_minor_ver}" == "${prev_minor}" ]; then
					found_prev_tag=true
				fi
			fi
		done
	elif [ "${patch_ver}" != 0 ]; then
		prev_patch=$(( patch_ver - 1))
		for tag in ${existing_tags}; do
			check_prefix=$(echo "${tag}" | grep 'ena-manifest')
			if [ "${check_prefix}" != "" ]; then
				tag_major_ver=$(echo "${tag}" | cut -d '.' -f 1 | cut -d '/' -f 2)
				tag_minor_ver=$(echo "${tag}" | cut -d '.' -f 2)
				tag_patch_ver=$(echo "${tag}" | cut -d '.' -f 3)
				if [ "${is_dev_version}" != "" ]; then
					tag_patch_ver=$(echo "${tag_patch_ver}" | cut -d '-' -f 1)
				fi
				if [ "${tag_major_ver}" == "${major_ver}" ] && [ "${tag_minor_ver}" == "${minor_ver}" ] && [ "${tag_patch_ver}" == "${prev_patch}" ]; then
					found_prev_tag=true
				fi
			fi
		done
	elif [ "${minor_ver}" == 0 ]; then
		prev_major=$(( major_ver - 1))
		for tag in ${existing_tags}; do
			check_prefix=$(echo "${tag}" | grep 'ena-manifest')
			if [ "${check_prefix}" != "" ]; then
				tag_major_ver=$(echo "${tag}" | cut -d '.' -f 1)
				if [ "${tag_major_ver}" == "${prev_major}" ]; then
					found_prev_tag=true
				fi
			fi
		done
	fi


	if [ ${found_prev_tag} = false ]; then
		echo "Invalid version $manifest_version. Expected parent version not found"
		return_code=1
	fi
}

function version_tag {
	echo "Creating git tag: ${new_tag}"
	local git_hash=""
	local commit_info=""

	git config --global user.email "do-not-reply@example.com"
	git config --global user.name "Sys_orch_github"

	git_hash=$(git rev-parse --short HEAD)
	commit_info=$(git log --oneline | grep "${git_hash}")

	git tag -a "$new_tag" -m "Tagged by Sys_orch_github. COMMIT:${commit_info}"

	echo "Tags including new tag:"
	git tag -n

	git push origin "$new_tag"
}

check_flag="$1"
if [ "${check_flag}" = "check" ]; then
	tag_check
	new_version_check
elif [ "${check_flag}" = "tag" ]; then
	if [ "${is_dev_version}" != "" ]; then
		echo "Dev verison, skip tagging"
	else
		version_tag
	fi
else
	# Default behiour is to just run a check if input not set
	tag_check
	new_version_check
fi

exit $return_code
