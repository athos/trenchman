# Trenchman
[![release](https://github.com/athos/trenchman/actions/workflows/release.yaml/badge.svg)](https://github.com/athos/trenchman/actions/workflows/release.yaml)
[![test](https://github.com/athos/trenchman/actions/workflows/test.yaml/badge.svg)](https://github.com/athos/trenchman/actions/workflows/test.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/athos/trenchman)](https://goreportcard.com/report/github.com/athos/trenchman)

A standalone nREPL/prepl client written in Go and heavily inspired by [Grenchman](https://github.com/technomancy/grenchman)

Trenchman is a standalone nREPL/prepl client, which means that it can be used as an ordinary REPL without having to make it cooperate with an editor or any other development tool.
Unlike ordinary Clojure REPLs, it starts up instantly as it just connects to a running nREPL/prepl server, eliminating the overhead of launching a JVM process and bootstrapping Clojure for every startup.

## Features

- Fast startup
- Written in Go and runs on various platforms
- Support for nREPL and prepl
- Works as a language-agnostic nREPL client

## Table of Contents

- [Trenchman](#trenchman)
  - [Features](#features)
  - [Table of Contents](#table-of-contents)
  - [Installation](#installation)
    - [Homebrew (macOS and Linux)](#homebrew-macos-and-linux)
    - [Scoop installer (Windows, Intel and AMD)](#scoop-installer-windows-intel-and-amd)
    - [Manual Install](#manual-install)
  - [Usage](#usage)
    - [Connecting to a server](#connecting-to-a-server)
      - [Port file](#port-file)
      - [Retry on connection](#retry-on-connection)
    - [Evaluation](#evaluation)
      - [Evaluating an expression (`-e`)](#evaluating-an-expression--e)
      - [Evaluating a file (`-f`)](#evaluating-a-file--f)
      - [Calling `-main` for a namespace (`-m`)](#calling--main-for-a-namespace--m)
  - [License](#license)

## Installation

### Homebrew (macOS and Linux)

To install Trenchman via [Homebrew](https://brew.sh/), run the following command:

```console
brew install athos/tap/trenchman
```

To upgrade:

```console
brew upgrade trenchman
```

### Scoop installer (Windows, Intel and AMD)

To install Trenchman via [Scoop](https://scoop.sh), run following commands:

```console
scoop bucket add scoop-clojure https://github.com/littleli/scoop-clojure
scoop install trenchman
```

To upgrade use:

```console
scoop update *
```

Note: To install on ARM architecture you have to use manual install.

### Manual Install

Pre-built binaries are available for linux, macOS and Windows on the [releases](https://github.com/athos/trenchman/releases) page.

If you have the Go tool chain installed, you can build and install Trenchman by the following command:

```console
go install github.com/athos/trenchman/cmd/trench@latest
```

Trenchman does not have `readline` support at this time. If you want to use features like line editing or command history, we recommend using [`rlwrap`](https://github.com/hanslub42/rlwrap) together with Trenchman.

## Usage

```
usage: trench [<flags>] [<args>...]

Flags:
      --help                    Show context-sensitive help (also try --help-long and --help-man).
  -p, --port=PORT               Connect to the specified port.
      --port-file=FILE          Specify port file that specifies port to connect to. Defaults to .nrepl-port.
  -P, --protocol=nrepl          Use the specified protocol. Possible values: n[repl], p[repl]. Defaults to nrepl.
  -s, --server=[(nrepl|prepl)://]host[:port]|nrepl+unix:path
                                Connect to the specified URL (e.g. prepl://127.0.0.1:5555, nrepl+unix:/foo/bar.socket).
      --retry-timeout=DURATION  Timeout after which retries are aborted. By default, Trenchman never retries connection.
      --retry-interval=1s       Interval between retries when connecting to the server.
  -i, --init=FILE               Load a file before execution.
  -e, --eval=EXPR               Evaluate an expression.
  -f, --file=FILE               Evaluate a file.
  -m, --main=NAMESPACE          Call the -main function for a namespace.
      --init-ns=NAMESPACE       Initialize REPL with the specified namespace. Defaults to "user".
  -C, --color=auto              When to use colors. Possible values: always, auto, none. Defaults to auto.
      --debug                   Print debug information
      --version                 Show application version.

Args:
  [<args>]  Arguments to pass to -main. These will be ignored unless -m is specified.
```

### Connecting to a server

One way to connect to a running server using Trenchman is to specify the server URL with the `-s` (`--server`) option. For example, the following command lets you connect to an nREPL server listening on `localhost:12345`:

```console
trench -s nrepl://localhost:12345
```

Trenchman 0.4.0+ can also establish an nREPL connection via UNIX domain socket.
To do so, specify the `--server` option with the `nrepl+unix:` scheme
followed by the address path of the socket:

```console
trench -s nrepl+unix:/foo/bar.socket
```

In addition to nREPL, Trenchman supports the prepl protocol as well.
To connect to a server via prepl, use the `prepl://` scheme instead of `nrepl://`:

```console
trench -s prepl://localhost:5555
```

Also, the connecting port and protocol can be specified with dedicated options:

- port: `-p`, `--port=PORT`
- protocol: `-P`, `--protocol=(nrepl|prepl)`

If you omit the protocol or server host, Trenchman assumes that the following default values are specified:

- protocol: `nrepl`
- server host: `127.0.0.1`

So, in order to connect to `nrepl://127.0.0.1:12345`, you only have to do:

```console
trench -p 12345
```

rather than `trench -s nrepl://127.0.0.1:12345`.

If you omit the port number, Trenchman will read it from a port file, as described in the next section.

#### Port file

A *port file* is a file that only contains the port number that the server is listening on.
Typical nREPL servers generate a port file named `.nrepl-port` at startup.

Trenchman tries to read the port number from a port file if the connecting port is not specified explicitly. By default, Trenchman will read `.nrepl-port` for nREPL connection and `.prepl-port` for prepl connection.

So, the following example connects to `nrepl://127.0.0.1:12345`:

```console
$ cat .nrepl-port
12345
$ trench
```

If you'd rather use another file as a port file, specify it with the `--port-file` option:

```console
$ cat my-port-file
3000
$ trench --port-file my-port-file
```

#### Retry on connection

When connecting to a server that is starting up, it's useful to be able to automatically retry the connection if it fails.

The `--retry-timeout` and `--retry-interval` options control connection retries.
`--retry-timeout DURATION` specifies the amount of time before connection retries are aborted and `--retry-interval DURATION` specifies the time interval between each retry (`DURATION` can be specified in the format accepted by [Go's duration parser](https://pkg.go.dev/time#ParseDuration), like `500ms`, `10s` or `1m`).

For example, the following command will retry the connection every 5 seconds for up to 30 seconds:

```console
trench --retry-timeout 30s --retry-interval 5s
```

If the connection fails after retrying the connection until the timeout, Trenchman will print the error and exit.

If `--retry-timeout` is not specified, Trenchman will not retry the connection.

### Evaluation

By default, Trenchman starts a new REPL session after the connection is established:

```console
$ trench
user=> (println "Hello, World!")
Hello, World!
nil
user=>
```

To exit the REPL session, type `Ctrl-D` or `:repl/quit`.

In addition to starting a REPL session, Trenchman provides three more
evaluation modes (`-e`/`-f`/`-m`).

#### Evaluating an expression (`-e`)

If the `-e` option is specified with an expression, Trenchman evaluates that expression:

```console
$ trench -e '(println "Hello, World!")'
Hello, World!
$
```

Trenchman will print the evaluation result if the given expression evaluates to a non-`nil` value:

```console
$ trench -e '(map inc [1 2 3])'
(2 3 4)
$
```

#### Evaluating a file (`-f`)

With the `-f` option, you can load (evaluate) the specified file:

```console
$ cat hello.clj
(println "Hello, World!")
$ trench -f hello.clj
Hello, World!
$
```

Note that the specified file path is interpreted as one from the client's working directory.
The client will send the entire content of the file to the server once the connection is established.

If `-` is specified as the input file, the input code will be read from stdin:

```console
$ echo '(println "Hello, World!")' | trench -f -
Hello, World!
$
```

#### Calling `-main` for a namespace (`-m`)

With the `-m` option, you can call the `-main` function for the specified namespace:

```console
$ cat src/hello/core.clj
(ns hello.core)

(defn -main []
  (println "Hello, World!"))
$ trench -m hello.core
Hello, World!
```

Note that the file for the specified namespace must be on the server-side classpath.

## License

Copyright (c) 2021 Shogo Ohta

Distributed under the MIT License. See LICENSE for details.
