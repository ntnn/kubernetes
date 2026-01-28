#!/usr/bin/env bash

drop_replace() {
    local gomod="$1"

    if [[ -z "$gomod" ]]; then
        echo "Missing go.mod argument"
        return 1
    fi
}

drop_replace go.mod
find staging -type f -name go.mod | while read -r gomod; do
    drop_replace "$gomod"
done
