# Changelog

## [0.1.6]

### Changed

- Built with Go 1.26.8, up from 1.26.5, which closes four reachable stdlib
  vulnerabilities (GO-2026-6218 `net/url`, GO-2026-6090 `crypto/tls`,
  GO-2026-5972 `encoding/asn1`, GO-2026-5026 `net/http`).
- Updated every flokiorg dependency to its current release. `google.golang.org/grpc` moved to the
  current release, taking govulncheck from two reachable findings to none.
- The release now publishes a multi-arch container image to
  `ghcr.io/flokiorg/grpc-miner`, and every push to the default branch publishes an
  `:edge` image.

### Changed

- Built with Go 1.26.5. (#3)

## [0.1.5-beta]

### Fixes

- **`mining/miner_test.go`**: `TestMineWithIssue` used a mock whose `SubmitNonce` always returns an error, but the test asserted `acceptedBlocks == 1` — an outcome impossible given the mock, and a copy-paste leftover from the adjacent success-path test. Fixed the expected value to `0`.
- Two `context.WithTimeout` cancel-function leaks in `TestStartMining` and `TestMineWithIssue`.

### CI

- Added `.github/workflows/ci.yaml`: runs `go build`, `go vet`, and `go test` on push to `main` and on pull requests.

## [0.1.4-beta]

Dependency-only update: brings in `go-flokicoin` v0.26.0-alpha, plus a security fix. No source code changes in this repo.

See the [go-flokicoin v0.26.0-alpha](https://github.com/flokiorg/go-flokicoin/releases/tag/v0.26.0-alpha) release notes for what's new upstream.

### Dependency Security

- Bumped `google.golang.org/grpc` from `v1.76.0` to `v1.79.3`, closing GHSA-p77j-4mvh-x3m3 (CVSS 9.1, gRPC `:path` authorization-bypass). gminer only dials out to a mining pool as a client and never runs its own gRPC server, so this wasn't exploitable here.

## [0.1.3-beta]

### Dependency Updates

- Updated dependencies to align with `go-flokicoin v0.25.13-alpha`.
- Routine `go mod tidy` cleanup.

## [0.1.2-beta]

### Dependency Updates

- Updated `github.com/flokiorg/go-flokicoin` to `v0.25.13-alpha`, which includes the TestNet4 P2P port correction.
- Routine `go mod tidy` cleanup.

## [0.1.1-beta]

### Dependencies
- Updated `go-flokicoin` → **v0.25.7-beta**

#### Core
- Bumped `VERSION` from **0.1.0-alpha** → **0.1.1-beta**

## [0.1.0-alpha]

- This is a **pre-release** for testing and feedback.
- Developers and early adopters are encouraged to **report issues**.
