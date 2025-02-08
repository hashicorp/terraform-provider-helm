#!/bin/bash
set -e
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


function get_latest_version() { 
    if output=$(curl -s -f https://api.github.com/repos/hashicorp/terraform/git/refs/tags); then
        echo "$output" | \
            jq ".[] | .ref | split(\"/\") | .[2] | select(. | startswith(\"$1\"))" | \
                sort -V -r | head -1 
    else
        echo "Error: Failed to connect to GitHub API" >&2
        exit 1
    fi
}

# Call get_latest_version for each version and handle errors
v1_0=$(get_latest_version v1.0) || exit 1
v1_3=$(get_latest_version v1.3) || exit 1
v1_5=$(get_latest_version v1.5) || exit 1
v1_7=$(get_latest_version v1.7) || exit 1
v1_9=$(get_latest_version v1.9) || exit 1

# Construct the matrix
echo "matrix=[$v1_0, $v1_3, $v1_5, $v1_7, $v1_9]" >> "$GITHUB_OUTPUT"