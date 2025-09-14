#!/usr/bin/env bash
#
# This script syncs the ACE headers (which are from a Bazel dependency) to the ace/include/ace directory.
# This directory is gitignored, but running this allows gopls to find the headers for cgo.
# (And it can then be built with the plain Go toolchain without Bazel.)

set -euo pipefail

echo >&2 "Fetching ACE headers from Bazel external repository..."

bazel build @ace//:libace
mkdir -p ace/include/ace

echo >&2 "Copying them to ace/include/ace/..."
# we need to get the canonical name for the ace repo
# so, use the special "" representing the root repo (this project)
ace_canonical_name=$(bazel mod dump_repo_mapping "" | jq -r '.["ace"]')
ace_header_path="bazel-bueno/external/${ace_canonical_name}/ace"
rsync -av --delete "${ace_header_path}/" ace/include/ace/

echo >&2 "Done!"
