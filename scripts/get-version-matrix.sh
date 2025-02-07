#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


function get_latest_version() { 
    if output=$(curl -s -f https://api.github.com/repos/hashicorp/terraform/git/refs/tags); then
        echo "$output" | \
            jq ".[] | .ref | split(\"/\") | .[2] | select(. | startswith(\"$1\"))" | \
                sort -V -r | head -1 
    else
        echo "Error: Failed to connect to GitHub API" >&2
        return 1
    fi
}

echo "matrix=[$(get_latest_version v1.0), $(get_latest_version v1.3), $(get_latest_version v1.5), $(get_latest_version v1.7), $(get_latest_version v1.9)]" >> "$GITHUB_OUTPUT"
