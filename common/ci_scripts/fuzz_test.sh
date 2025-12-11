#!/usr/bin/env bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Review each source code folder for fuzz tests and run them
FUZZ_TIME=${2:-60s}
homeDir=$(pwd)
sourceCodeDir="${homeDir}/${1}"
sourceCodePkgs=$(ls "${sourceCodeDir}")
anyTestFailed=0

for pkg in ${sourceCodePkgs}; do
	cd "${sourceCodeDir}"/"${pkg}" || exit
	# Count test files in the directory
	checkTestFile=$(find . -maxdepth 1 -name '*_test.go' | wc -l)
	if [ "${checkTestFile}" -ne "0" ]; then
		testFiles=$(ls)
		for testFile in ${testFiles}; do
			# Skip testdata directory
			if [ "$testFile" != "testdata" ]; then
				# Count fuzz tests in the test file
				fuzzTestCount=$(grep 'func Fuzz' "${testFile}" -c1)
				if [ "${fuzzTestCount}" -ne "0" ]; then
					echo "${fuzzTestCount}" fuzz tests found in "${testFile}"
					# Extract fuzz test function names
					checkFuzzTest=$(grep 'func Fuzz' "${testFile}" | cut -d '(' -f 1 | cut -d ' ' -f 2)
					for fuzzTest in ${checkFuzzTest}; do
						echo running "${fuzzTest}" test case
						exitStatus=0
						if [ "${LOG_FUZZ_RESULTS}" == "true" ]; then
							echo "Write test output to file"
							logFile="${homeDir}/fuzz_${pkg}_${fuzzTest}.log"
							go test -fuzz "${fuzzTest}" -fuzztime "${FUZZ_TIME}" > "${logFile}" 2>&1
							exitStatus=$?
							echo "Output written to ${logFile}"
						else
							go test -fuzz "${fuzzTest}" -fuzztime "${FUZZ_TIME}"
							exitStatus=$?
						fi

						if [ $exitStatus -ne 0 ]; then
							echo "Fuzz test ${fuzzTest} in package ${pkg} FAILED"
							anyTestFailed=1
						fi
					done
					echo
				fi
			fi
		done
	fi
	cd "${homeDir}" || exit
done

if [ $anyTestFailed -ne 0 ]; then
	echo "One or more fuzz tests failed."
	exit 1
else
	echo "All fuzz tests passed."
	exit 0
fi
