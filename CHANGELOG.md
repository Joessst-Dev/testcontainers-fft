# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `Run` — start the fulfillmenttools emulator as a Testcontainers container, waiting on a
  token-free `GET /api/status` `200`.
- `Container` accessors: `BaseURL`, `MustBaseURL`, `HostPort` (embeds
  `testcontainers.Container`).
- Options: `WithSeed`, `WithSeedFS`, `WithVerbose`, `WithPubSubHost`.
- `RunWithPubSub` — start the emulator wired to a Google Pub/Sub emulator sidecar on a
  shared network, with `Stack.PubSubEndpoint` and `Stack.Terminate`.
- `DefaultImage` pinned to a tested emulator release.
