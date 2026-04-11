# Versioning

Jumpgate uses calendar versioning.

- `VERSION` stores the current release number in `YYYY.MM.PATCH` format.
- Git tags use a `v` prefix, for example `v2026.04.0`.
- The deployed service version adds the short commit hash, for example `2026.04.0+04fc78e`.
- `make release-prepare` computes the next monthly patch version from existing tags, creates `release/<version>` from `origin/main`, updates `VERSION`, pushes the branch, and opens the release PR.
- `make release-publish` runs after that PR merges. It tags the merged commit on `main` and creates the GitHub Release from `RELEASE_NOTES.md`.

Normal local builds report the release number from `VERSION` plus the current short git hash when git metadata is available. Docker source builds do the same when built from a normal git checkout.
