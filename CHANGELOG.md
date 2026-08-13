# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project uses
[semantic versioning](https://semver.org/spec/v2.0.0.html).

## 0.2.0

### Added

- Release builds for linux/riscv64, darwin/amd64, windows/amd64,
  windows/arm64, freebsd/amd64, and freebsd/arm64, in addition to the existing
  linux/amd64, linux/arm64, and darwin/arm64. `install.sh` now installs the
  darwin/amd64, linux/riscv64, and freebsd builds too; Windows builds are
  published but must be downloaded manually, since `install.sh` is a POSIX
  shell script.

### Fixed

- `-selfupdate` now requests `router-colorizer_windows_<arch>.exe`, matching
  the `.exe` suffix GoReleaser adds to Windows release assets. Previously it
  would have looked for an asset without the suffix and always reported no
  build available.

## 0.1.0

### Added

- Initial Go port of [App::RouterColorizer][perl] 1.242880, colorizing output
  from Arista EOS, Cisco IOS, Juniper Junos, VyOS, and Ciena devices.
- `colorizer.FormatText`, which colorizes a string or any fragment of a stream.
- `colorizer.FormatTextParallel`, which returns what `FormatText` returns while
  dividing the text at line endings across goroutines. `Filter` uses it for
  blocks large enough to be worth dividing, which excludes everything a live
  session produces, so interactive latency is unchanged.
- `colorizer.Filter`, which colorizes an `io.Reader` onto an `io.Writer` and
  writes out a partial line once it has gone idle, so that interactive prompts
  are not held back.
- `router-colorizer` command, a standard input to standard output filter, with
  `-flush-delay`, `-version`, and `-selfupdate`.
- Release builds for linux/amd64, linux/arm64, and darwin/arm64, published via
  GoReleaser on every `v*` tag, plus `install.sh` for installing the latest
  release with `curl | sh`.
- Test suite sharing its golden files with the Perl implementation, plus unit
  tests, a property fuzz target, and a benchmark.
- Recognition of `et-` Junos interface names, and of a channelized port's
  channel and logical unit, as in `et-0/0/1:0.0`. The Perl original colorizes
  neither.

### Fixed

Three defects in the Perl original are corrected here. None of them changes the
colorization of any captured output in the test suite.

- Nested colorization now restores the outer color in every case, rather than
  only when the inner reset happens to be surrounded by spaces. This affects
  re-colorizing already-colorized text.
- A Ciena alarm severity may now be padded with whitespace on either side. The
  Perl allows leading whitespace only for `critical`, `major`, `minor`, and
  `warning`, and trailing whitespace only for `info`, so an alarm padded on
  both sides is not colorized at all.
- Junos interface names and the Cisco `(connected)` suffix no longer accept a
  stray leading colon, which the Perl allows by writing `(:?` where `(?:` was
  meant.

[perl]: https://github.com/jmaslak/App-RouterColorizer
