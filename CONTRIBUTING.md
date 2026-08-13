# Contributing

You want to help? That's great!

## Ground rules

- **Don't break the public interface.** You can add to it, but code that
  already depends on `colorizer.FormatText` and `colorizer.Filter` should keep
  working.
- **Don't change the colors of existing output without saying so.** People
  learn what their addresses and their interface lines look like. Recoloring
  them is a user-visible change, not a refactor. The address hash in
  `colorizer/ip.go` is pinned by a test for this reason.
- **Bug reports** are welcome however you want to send them — GitHub issue
  preferred, email to <jmaslak@antelope.net> is fine. Security issues should
  be reported privately by email first, so there is a chance to fix them
  before they are public.
- **Code changes** are welcome however you want to send them, GitHub pull
  request preferred.
- **You will be credited** unless you ask otherwise. If the credit is wrong,
  send a note or a PR.

## Things that would help

- Support for other vendors. The rules were written against Arista, Cisco,
  Juniper, VyOS, and Ciena; there are a lot of other CLIs out there.
- Test cases from vendors that are already supported but whose output has
  drifted.
- Documentation and examples.
- If something is missing, confusing, or does not work the way you expect,
  say so. The tool makes sense to its author and their way of working; if it
  does not make sense to you, you are probably not the only one.

## Adding a rule

To colorize a line the tool does not recognize yet:

1. Add a rule to the table for that vendor — `arista.go`, `junos.go`, or
   `ciena.go`. Put a specific rule *above* the general rule it should win
   against; rule order is what decides ties. See "How it works" in the README.
2. Add the line to `colorizer/testdata/<vendor>.input` and its expected
   colorization to the matching `.output` file, on the same line number. One
   line per test case.
3. Run the tests.

To see what a colorized line actually looks like as bytes:

```sh
printf 'your line here\n' | go run ./cmd/router-colorizer | cat -v
```

## Before sending a change

```sh
gofmt -l .          # should print nothing
go vet ./...
go test -race ./...
```

If your change touches line splitting, escape handling, or address matching,
also let the fuzzer run for a while:

```sh
go test -fuzz FuzzFormatText -fuzztime 2m ./colorizer
```

## Relationship to the Perl version

This is a port of [App::RouterColorizer][perl]. The Perl version has
been deprecated when the code was migrated from Perl to Go.

[perl]: https://github.com/jmaslak/App-RouterColorizer

By participating in this project you agree to abide by its
[Code of Conduct](CODE_OF_CONDUCT.md).
