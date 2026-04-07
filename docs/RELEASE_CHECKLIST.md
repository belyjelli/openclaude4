# Release checklist

Use this before tagging a release of the `openclaude` Go binary.

## Version and tags

- Use **semver** with a leading `v` on the git tag (for example `v0.2.0`), consistent with [goreleaser.yml](../goreleaser.yml) and Go module conventions.
- Confirm `main.version` and `main.commit` are injected via **ldflags** at build time (see [README.md](../README.md) local build example and `goreleaser.yml` `builds[].ldflags`).

## Build and artifacts

- Run `go test ./...`, `go vet ./...`, and CI-equivalent checks locally if you changed core paths.
- Cut the release with **GoReleaser**, for example `goreleaser release --clean` (or the project’s CI workflow), targeting GitHub Releases for `gitlawb/openclaude4`.
- Verify **`checksums.txt`** on the release when publishing download instructions.

## Changelog

- GoReleaser generates a changelog from commits ([goreleaser.yml](../goreleaser.yml) `changelog` — ascending sort, excludes commits matching `^docs:`, `^steps:`, `^test:`).
- For user-facing releases, consider **editing release notes** on GitHub to highlight breaking changes, new flags, and security-relevant behavior; the auto changelog is a starting point only.

## Security

- Point users at [SECURITY.md](./SECURITY.md) for CLI behavior (workspace boundary, dangerous tools, network defaults).
- **Reporting vulnerabilities:** use GitHub **Security → Private vulnerability reporting** (or advisories) when enabled for this repository; do not post exploit details in public issues first.
