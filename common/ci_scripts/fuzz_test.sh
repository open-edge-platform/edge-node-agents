#!/usr/bin/env bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Review each source code folder for fuzz tests and run them
FUZZ_TIME=${2:-60s}
homeDir=$(pwd)
sourceCodeDir=${homeDir}/${1}
sourceCodePkgs=$(ls ${sourceCodeDir})
anyTestFailed=0

for pkg in ${sourceCodePkgs}; do
	cd ${sourceCodeDir}/${pkg} || exit
	checkTestFile=$(ls | grep -c '_test.go')
	if [ ${checkTestFile} -ne "0" ]; then
		testFiles=$(ls)
		for testFile in ${testFiles}; do
			if [ $testFile != "testdata" ]; then
				fuzzTestCount=$(grep 'func Fuzz' ${testFile} -c1)
				if [ ${fuzzTestCount} -ne "0" ]; then
					echo ${fuzzTestCount} fuzz tests found in ${testFile}
					checkFuzzTest=$(grep 'func Fuzz' ${testFile} | cut -d '(' -f 1 | cut -d ' ' -f 2)
					for fuzzTest in ${checkFuzzTest}; do
						echo running ${fuzzTest} test case
						logFile="${homeDir}/fuzz_${pkg}_${fuzzTest}.log"
						go test -fuzz ${fuzzTest} -fuzztime "${FUZZ_TIME}" > "${logFile}" 2>&1
						exitStatus=$?
						echo "Output written to ${logFile}"
						
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
	cd ${homeDir} || exit
done

if [ $anyTestFailed -ne 0 ]; then
	echo "One or more fuzz tests failed."
	exit 1
else
	echo "All fuzz tests passed."
	exit 0
fi
