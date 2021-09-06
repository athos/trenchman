# Changelog
All notable changes to this project will be documented in this file. This change log follows the conventions of [keepachangelog.com](http://keepachangelog.com/).

## [Unreleased]
### Added
- New options `--retry-timeout` / `--retry-interval` to control connection retries [#7](https://github.com/athos/trenchman/pull/7)

### Changed
- The `nrepl` and `prepl` clients now take a `ConnBuilder` on construction instead of a pair of `Host` and `Port`.

## [v0.2.0] - 2021-08-24
### Added
- New option `--init-ns` to specify initial REPL namespace [#3](https://github.com/athos/trenchman/pull/3)
- New option `--init` to load a file before execution [#4](https://github.com/athos/trenchman/pull/4)

### Changed
- Add newlines to each prelude message to work around a bug in ClojureScript's prepl server [#5](https://github.com/athos/trenchman/pull/5)
- Command-line args can now be passed to -main [#6](https://github.com/athos/trenchman/pull/6)

## [v0.1.1] - 2021-08-14
- Same as v0.1.0, but fixes a Homebrew release bug

## [v0.1.0] - 2021-08-14
- First release

[Unreleased]: https://github.com/athos/trenchman/compare/v0.2.0...HEAD
[v0.2.0]: https://github.com/athos/trenchman/compare/v0.1.1...v0.2.0
[v0.1.1]: https://github.com/athos/trenchman/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/athos/trenchman/releases/tag/v0.1.0
