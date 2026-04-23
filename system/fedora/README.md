Fedora RPM packaging
====================

This directory contains everything needed to build x86_64 RPMs of algernon for
Fedora 43 and Fedora 44.

Files
-----

- `algernon.spec` — RPM spec, targets Fedora 43 and 44. Installs the binary,
  `mdview`, the production systemd unit (`algernon.service`), the development
  unit as a `.example` file under `/usr/share/doc/algernon/`, the logrotate
  snippet, the sample `serverconf.lua`, desktop integration, the man page and
  the `samples/` tree.
- `build.sh` — produces a source tarball from the git checkout, builds an SRPM
  and then runs `mock` once per target release. Falls back to plain `rpmbuild`
  on the current host when `mock` is unavailable (the other Fedora release is
  skipped with a warning). Output lands in `dist/`.

Building
--------

On a Fedora host with `rpm-build`, `rpmdevtools` and `mock` installed, and the
current user in the `mock` group:

    ./build.sh

RPMs are written to:

    dist/fedora-43-x86_64/
    dist/fedora-44-x86_64/

Without mock, `build.sh` falls back to `rpmbuild --rebuild` on the host and
only produces an RPM for the host's own Fedora release.

Building manually for a single release
--------------------------------------

    # Create the source tarball and SRPM
    ./build.sh   # or extract the relevant commands from it
    mock -r fedora-44-x86_64 --rebuild dist/srpm/algernon-*.src.rpm

Notes
-----

- The spec uses the vendored dependencies committed to `vendor/`, so the mock
  chroot does not need network access during the build.
- `ExclusiveArch: x86_64` matches the current Arch PKGBUILD; drop or extend it
  if other architectures are tested.
- The version string in `algernon.spec` must track `version.sh`. Update both
  together when cutting a release.
