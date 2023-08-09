#!/bin/sh

min_ulimit=512
ulimit=$(ulimit -n 2> /dev/null || echo "not found")

# Use "sysctl kern.maxfiles" for macOS
kernmax=$(sysctl kern.maxfiles 2> /dev/null || echo "not found")
kernmax_present=$(echo "$kernmax" | grep -qE '^kern\.maxfiles:'; echo $?)
kernmax_valid_format=$(echo "$kernmax" | grep -qE '^kern\.maxfiles: [0-9]+$'; echo $?)

# Check if $kernmax is present and not in the valid format
if [ "$kernmax_present" -eq 0 ] && [ "$kernmax_valid_format" -ne 0 ]; then
    echo 'ERROR:'
    echo "  'sysctl kern.maxfiles' did not return a valid number"
    exit 1
fi

# If kernmax is present and has a valid format, set that as the ulimit
if [ "$kernmax_present" -eq 0 ]; then
    kernmax_value=$(echo "$kernmax" | awk '{print $2}')
    ulimit="$kernmax_value"
fi

# If we have no ulimit number by now, exit with an error message
if [ "$ulimit" = "not found" ]; then
    echo 'NOTE:'
    echo "  'ulimit -n' could not be run."
    echo "  'ulimit -n' returns the number of files that can be open at the same time."
    echo "  The '--autorefresh' flag will not work if the ulimit is too low."
    exit 1
fi

if [ "$ulimit" -lt "$min_ulimit" ]; then
    echo 'NOTE:'
    echo "  'ulimit -n' on your system returns $ulimit."
    echo "  'ulimit -n' returns the number of files that can be open at the same time."
    echo "  $ulimit is too low to be able to use the '--autorefresh' flag."
    echo "  Increase the ulimit on your system to at least $min_ulimit to be able to use it."
    exit 1
fi

echo 'Try editing the markdown file in "samples" directory and see the'
echo 'results instantly in the browser at http://localhost:3000/'

./algernon --dev --conf serverconf.lua --dir samples --httponly --debug --autorefresh --bolt --server "$@"
