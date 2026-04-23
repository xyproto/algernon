#!/bin/bash
# Build x86_64 RPMs of algernon for Fedora 43 and Fedora 44.
#
# Preferred path: mock (clean chroot, cross-release). Requires the user to
# be in the "mock" group.
#
# Fallback: plain rpmbuild on the current host. Only the host's own Fedora
# release gets built; the other release is skipped with a warning.
#
# Output: RPMs under system/fedora/dist/<release>/ plus the SRPM under
# system/fedora/dist/srpm/.

set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$here/../.." && pwd)
spec="$here/algernon.spec"

version=$(rpmspec -q --qf '%{version}\n' "$spec" | head -n1)
name=algernon
releases=(fedora-43-x86_64 fedora-44-x86_64)

dist="$here/dist"
src="$dist/src"
srpm_dir="$dist/srpm"
mkdir -p "$src" "$srpm_dir"

# vendor/ is committed so `git archive` alone captures everything mock/rpmbuild
# needs for an offline build.
tarball="$src/${name}-${version}.tar.gz"
echo "Creating $tarball"
rm -f "$tarball"
git -C "$repo" archive --format=tar.gz --prefix="${name}-${version}/" -o "$tarball" HEAD

# Build the SRPM once, reuse for every target.
rpmbuild -bs "$spec" \
    --define "_sourcedir $src" \
    --define "_srcrpmdir $srpm_dir" \
    --define "_topdir $dist/rpmbuild" >/dev/null
srpm=$(ls -1t "$srpm_dir"/${name}-${version}-*.src.rpm | head -n1)
echo "Built SRPM: $srpm"

# Decide how to build the binary RPMs.
use_mock=0
if command -v mock >/dev/null && id -nG | tr ' ' '\n' | grep -qx mock; then
    use_mock=1
fi

if [ $use_mock -eq 1 ]; then
    for r in "${releases[@]}"; do
        out="$dist/$r"
        mkdir -p "$out"
        echo "=== mock rebuild on $r ==="
        mock -r "$r" --resultdir="$out" --rebuild "$srpm"
    done
else
    echo
    echo "mock is unavailable (not installed, or user not in the 'mock' group)."
    echo "Falling back to plain rpmbuild on the current host."
    host_rel=$(rpm -E '%{?fedora}')
    if [ -z "$host_rel" ]; then
        echo "Not on Fedora, cannot build binary RPMs without mock." >&2
        exit 1
    fi
    host_target="fedora-${host_rel}-x86_64"
    out="$dist/$host_target"
    mkdir -p "$out"
    rpmbuild --rebuild "$srpm" \
        --define "_topdir $dist/rpmbuild" \
        --define "_rpmdir $out" \
        --define "_rpmfilename %%{NAME}-%%{VERSION}-%%{RELEASE}.%%{ARCH}.rpm"
    for r in "${releases[@]}"; do
        [ "$r" = "$host_target" ] && continue
        echo "Skipping $r: needs mock for a clean chroot build."
    done
fi

echo
echo "Built RPMs:"
find "$dist" -name "*.rpm" -not -name "*.src.rpm" | sort
