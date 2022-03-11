#!/bin/bash

function get_latest_version() { 
    curl -s https://api.github.com/repos/hashicorp/terraform/git/refs/tags | \
        jq ".[] | .ref | split(\"/\") | .[2] | select(. | startswith(\"$1\"))" | \
            sort -V -r | head -1 
}

echo "::set-output name=matrix::[$(get_latest_version v0.12), $(get_latest_version v0.13), $(get_latest_version v0.14), $(get_latest_version v0.15), $(get_latest_version v1.0)]"
