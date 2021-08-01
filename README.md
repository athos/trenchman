# Trenchman
[![test](https://github.com/athos/trenchman/actions/workflows/test.yaml/badge.svg)](https://github.com/athos/trenchman/actions/workflows/test.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/athos/trenchman)](https://goreportcard.com/report/github.com/athos/trenchman)

Trenchman is a standalone nREPL/prepl client written in Go, heavily inspired by [Grenchman](https://github.com/technomancy/grenchman).

## Features

- Fast startup
- Support for nREPL and prepl
- Written in Go and runs on various platforms

## Installation

If you have the Go tool chain installed, you can build and install Trenchman by the following command:

```sh
$ go install github.com/athos/trenchman/cmd/trench@latest
```

Trenchman does not have readline support at this time. If you want to use features like line editing or command history, we recommend using [`rlwrap`](https://github.com/hanslub42/rlwrap) together with Trenchman.

## Usage

(TODO)

## License

Copyright (c) 2021 Shogo Ohta

Distributed under the MIT License. See LICENSE for more details.
