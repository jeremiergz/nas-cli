## Copilot instructions for contributors

This project is a Go-based CLI that manages a personal NAS. The notes below capture project-specific architecture, developer workflows, and conventions an AI code assistant should follow to be productive.

- **Big picture**: The binary is a Cobra-based CLI. The command graph is built in `main.go` (commands are registered with `rootCmd.AddCommand(...)`). See [main.go](../main.go) and the central command helper in [internal/cmd/cmd.go](../internal/cmd/cmd.go).
- **Service singletons**: Shared services (console, SFTP, SSH) are global variables in `internal/service/service.go` and are initialized in `init()`. Use `svc.Console`, `svc.SFTP`, `svc.SSH` to access these singletons. Never re-initialize them; treat them as global resources. See [internal/service/service.go](../internal/service/service.go).
- **Command pattern**: Each CLI feature lives under `internal/cmd/<name>` and exposes `New() *cobra.Command`. Example: [internal/cmd/backup/backup.go](../internal/cmd/backup/backup.go) implements flags, subcommands, and helpers. Follow the same `New()` factory signature and register with `main.go`.
- **Configuration**: App settings use `sp13/viper`. Keys are defined in [internal/config/config.go](../internal/config/config.go). The configuration file is saved to `~/.nascliconfig` (YAML). Respect the viper keys (e.g. `nas.fqdn`, `ssh.client.privatekey`) when reading/writing config.
- **Build & release**: Use the `Makefile` targets: `make build`, `make test`, `make coverage`, `make release`. The `Makefile` embeds version info via `-ldflags` into `internal/config` (AppName, BuildDate, GitCommit, Version). See [Makefile](../Makefile) for multi-arch `build-all` targets.
- **Testing**: Tests run with `go test ./...` or `make test`. Coverage profile is written to `coverage/profile.cov`. Use the `coverage` or `coverage-html` Makefile targets for coverage reports.
- **Output & UI**: The project uses `pterm` for interactive output. Tests and code should not assume a TTY; use the `svc.Console` and command `OutOrStdout()` to capture output as other code does.
- **Error & lifecycle patterns**: `internal/cmd/cmd.go` wires a `PersistentPostRun` that disconnects `svc.SFTP` and prints debug info when `--debug` is set. Follow this lifecycle: avoid leaving connections open and respect `cmdutil.DebugMode` flags.
- **Common utilities**: Helper functions live under `internal/util` and `service/internal/*`. Reuse `internal/util/cmdutil`, `util/fsutil`, and `service` internals where appropriate.
- **Concurrency knobs**: CLI exposes `--max-concurrent-threads` in `internal/cmd/cmd.go` that maps to `cmdutil.MaxConcurrentGoroutines`. Respect this when adding goroutines; prefer controlled worker pools.
- **Third-party integrations**: The code interacts with SSH/SFTP and Plex. Configuration keys for these integrations live in [internal/config/config.go](../internal/config/config.go). Look for `KeyPlexAPIURL`, `KeyPlexAPIToken`, and SSH client keys.
- **File and path conventions**: Paths in config may contain `~` and are normalized in `internal/config/config.go` (e.g., known_hosts and private key). When accepting a path as input, follow the same tilde-expansion and `filepath.Abs` normalization.
- **Where to add code**: New commands -> `internal/cmd/<name>/...`. New reusable logic -> `internal/<area>` (e.g., `internal/media`, `internal/service`). Avoid adding broad globals; prefer adding initialization in the `service` package if it truly is a shared runtime service.
- **Quick examples**:
  - Register a new command: add package `internal/cmd/foo` with `func New() *cobra.Command` and in `main.go` call `rootCmd.AddCommand(foo.New())`.
  - Read a config value: `viper.GetString(internal/config.KeySSHClientPrivateKey)` — keys are constants in [internal/config/config.go](../internal/config/config.go).

If you want changes merged differently, or need more examples (e.g. how to implement a new SFTP-backed command), say which area you'd like expanded and I will iterate.
## Copilot instructions for contributors

This project is a Go-based CLI that manages a personal NAS. The notes below capture project-specific architecture, developer workflows, and conventions an AI code assistant should follow to be productive.

- **Big picture**: The binary is a Cobra-based CLI. The command graph is built in `main.go` (commands are registered with `rootCmd.AddCommand(...)`). See [main.go](main.go) and the central command helper in [internal/cmd/cmd.go].
- **Service singletons**: Shared services (console, SFTP, SSH) are global variables in `internal/service/service.go` and are initialized in `init()`. Use `svc.Console`, `svc.SFTP`, `svc.SSH` to access these singletons. Never re-initialize them; treat them as global resources.
- **Command pattern**: Each CLI feature lives under `internal/cmd/<name>` and exposes `New() *cobra.Command`. Example: `internal/cmd/backup/backup.go` implements flags, subcommands, and helpers. Follow the same `New()` factory signature and register with `main.go`.
- **Configuration**: App settings use `sp13/viper`. Keys are defined in `internal/config/config.go`. The configuration file is saved to `~/.nascliconfig` (YAML). Respect the viper keys (e.g. `nas.fqdn`, `ssh.client.privatekey`) when reading/writing config.
- **Build & release**: Use the `Makefile` targets: `make build`, `make test`, `make coverage`, `make release`. The `Makefile` embeds version info via `-ldflags` into `internal/config` (AppName, BuildDate, GitCommit, Version). See `Makefile` for multi-arch `build-all` targets.
- **Testing**: Tests run with `go test ./...` or `make test`. Coverage profile is written to `coverage/profile.cov`. Use the `coverage` or `coverage-html` Makefile targets for coverage reports.
- **Output & UI**: The project uses `pterm` for interactive output. Tests and code should not assume a TTY; use the `svc.Console` and command `OutOrStdout()` to capture output as other code does.
- **Error & lifecycle patterns**: `internal/cmd/cmd.go` wires a `PersistentPostRun` that disconnects `svc.SFTP` and prints debug info when `--debug` is set. Follow this lifecycle: avoid leaving connections open and respect `cmdutil.DebugMode` flags.
- **Common utilities**: Helper functions live under `internal/util` and `service/internal/*`. Reuse `internal/util/cmdutil`, `util/fsutil`, and `service` internals where appropriate.
- **Concurrency knobs**: CLI exposes `--max-concurrent-threads` in `internal/cmd/cmd.go` that maps to `cmdutil.MaxConcurrentGoroutines`. Respect this when adding goroutines; prefer controlled worker pools.
- **Third-party integrations**: The code interacts with SSH/SFTP and Plex. Configuration keys for these integrations live in `internal/config/config.go`. Look for `KeyPlexAPIURL`, `KeyPlexAPIToken`, and SSH client keys.
- **File and path conventions**: Paths in config may contain `~` and are normalized in `internal/config/config.go` (e.g., known_hosts and private key). When accepting a path as input, follow the same tilde-expansion and `filepath.Abs` normalization.
- **Where to add code**: New commands -> `internal/cmd/<name>/...`. New reusable logic -> `internal/<area>` (e.g., `internal/media`, `internal/service`). Avoid adding broad globals; prefer adding initialization in the `service` package if it truly is a shared runtime service.
- **Quick examples**:
  - Register a new command: add package `internal/cmd/foo` with `func New() *cobra.Command` and in `main.go` call `rootCmd.AddCommand(foo.New())`.
  - Read a config value: `viper.GetString(internal/config.KeySSHClientPrivateKey)` — keys are constants in `internal/config/config.go`.

If you want changes merged differently, or need more examples (e.g. how to implement a new SFTP-backed command), say which area you'd like expanded and I will iterate.
