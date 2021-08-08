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

### Connecting to a server

### Evaluation

Trenchman provides three evaluation modes as with the Clojure CLI.

#### Evaluating an expression

```sh
$ trench -e '(println "Hello, World!")'
Hello, World!
$
```

```sh
$ trench -e '(map inc [1 2 3])'
(2 3 4)
$
```

#### Evaluating a file

```sh
$ cat hello.clj
(println "Hello, World!")
$ trench -f hello.clj
Hello, World!
$
```

#### Calling `-main` for a namespace

```sh
$ trench -m foo.bar
```

## License

Copyright (c) 2021 Shogo Ohta

Distributed under the MIT License. See LICENSE for more details.
