#!/bin/bash -eu
# Copyright 2023 The Wuffs Authors.
#
# Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
# https://www.apache.org/licenses/LICENSE-2.0> or the MIT license
# <LICENSE-MIT or https://opensource.org/licenses/MIT>, at your
# option. This file may not be copied, modified, or distributed
# except according to those terms.
#
# SPDX-License-Identifier: Apache-2.0 OR MIT

# ----------------

# This script prints the CRC-32 checksum of the decompressed data of every file
# given as a command line argument: if a file then itself, if a directory then
# the files under it. It skips any non-compressed file found (non-compressed
# meaning not decodable by Wuffs' standard library).
#
# It is not perfect. It can have false positives and false negatives.
# Nonetheless, running it regularly (compiled against the in-development
# release/c/wuffs-unsupported-snapshot.c) can help detect regressions.

if [ ! -e wuffs-root-directory.txt ]; then
  echo "$0 should be run from the Wuffs root directory."
  exit 1
elif [ ! -e gen/bin/example-mzcat ]; then
  echo "Run \"./build-example.sh example/mzcat\" first."
  exit 1
elif [ ! -e gen/bin/example-crc32 ]; then
  echo "Run \"./build-example.sh example/crc32\" first."
  exit 1
fi

sources=$@
if [ $# -eq 0 ]; then
  sources=test/data
fi

# ----

handle() {
  # "Use $x for the exit code" is loosely based on
  # https://stackoverflow.com/a/16530815
  local x
  local o=$(((((gen/bin/example-mzcat <$1 2>/dev/null; echo $? >&3) |\
      gen/bin/example-crc32 >&4) 3>&1) | (read x; echo $x)) 4>&1)
  local a=($o)
  # ${a[0]} holds the exit code   of gen/bin/example-etc.
  # ${a[1]} holds the CRC-32 hash of gen/bin/example-etc's output.

  if [ ${a[0]} == 0 ]; then
    echo OK. ${a[1]} $1
  elif [ ${a[1]} != "00000000" ]; then
    echo BAD ${a[1]} $1
  fi
}

# ----

echo "# Generated by script/print-mzcat-checksums.sh"
for f in $sources; do
  if [ -f $f ]; then
    handle $f
  elif [ -d $f ]; then
    for g in `find $f -type f | LANG=C sort`; do
      handle $g
    done
  else
    echo "Could not open $f"
    exit 1
  fi
done