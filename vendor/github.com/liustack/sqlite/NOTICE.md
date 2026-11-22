# Notices

This repository holds code from several sources, each under its own license. This file says which license covers which path. It is not legal advice.

| Code | Written by | License | License file |
| --- | --- | --- | --- |
| Everything not named below | The liustack/sqlite Authors ([AUTHORS](AUTHORS)) | MIT | [LICENSE](LICENSE) |
| `engine/sqlite3.c`, `engine/sqlite3.h`: the SQLite amalgamation, unchanged | D. Richard Hipp and the SQLite developers | Public domain | The notice at the top of each file, and https://sqlite.org/copyright.html |
| `engine/libc/string/*.c` except `memcpy.c`, `engine/libc/math/*`, `engine/libc/internal/libm.h` and `engine/libc/COPYRIGHT`: musl at commit `7ada6dd` (2024-03-02), unchanged except `libm.h`, `memcmp.c` and `strcspn.c` (the patches in `engine/libc/patches/`) | Rich Felker and others (musl) | MIT | [engine/libc/COPYRIGHT](engine/libc/COPYRIGHT) |
| `replica/internal/s3/testdata/sigv4/`: the SigV4 test suite of aws-c-auth at commit `bdb7cef`, the cases S3 signs alike, unchanged | Amazon.com, Inc. or its affiliates | Apache-2.0 | [replica/internal/s3/testdata/sigv4/LICENSE](replica/internal/s3/testdata/sigv4/LICENSE) |
| `CODE_OF_CONDUCT.md`: the Contributor Covenant, version 2.1 | Coraline Ada Ehmke and the Contributor Covenant contributors | CC BY 4.0 | The attribution at the end of the file, and https://www.contributor-covenant.org/version/2/1/code_of_conduct/ |
| `assets/brand/sqlite-banner.png`, `assets/brand/sqlite-og.png`: the gopher in them, a redrawn Go gopher | Renee French (the Go gopher) | CC BY 4.0 | https://go.dev/blog/gopher and https://creativecommons.org/licenses/by/4.0/ |
