# Trenchman
[![test](https://github.com/athos/trenchman/actions/workflows/test.yaml/badge.svg)](https://github.com/athos/trenchman/actions/workflows/test.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/athos/trenchman)](https://goreportcard.com/report/github.com/athos/trenchman)

Trenchman is a standalone nREPL/prepl client written in Go, heavily inspired by [Grenchman](https://github.com/technomancy/grenchman).

Trenchman is a standalone nREPL/prepl client, which means that it can be used as an ordinary REPL without having to make it cooperate with an editor or any other development tool.
Unlike ordinary Clojure REPLs, it starts up instantly as it just connects to a running nREPL/prepl server, eliminating the overhead of launching a JVM process and bootstrapping Clojure for every startup.

## Features

- Written in Go and runs on various platforms
- Fast startup
- Support for nREPL and prepl
- Works as a language-agnostic nREPL client

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
  - [Connecting to a server](#connecting-to-a-server)
    - [Port file](#port-file)
  - [Evaluation](#evaluation)
    - [Evaluating an expression (`-e`)](#evaluating-an-expression--e)
    - [Evaluating a file (`-f`)](#evaluating-a-file--f)
    - [Calling `-main` for a namespace (`-m`)](#calling--main-for-a-namespace--m)

## Installation

If you have the Go tool chain installed, you can build and install Trenchman by the following command:

```sh
$ go install github.com/athos/trenchman/cmd/trench@latest
```

Trenchman does not have readline support at this time. If you want to use features like line editing or command history, we recommend using [`rlwrap`](https://github.com/hanslub42/rlwrap) together with Trenchman.

## Usage

### Connecting to a server

One way to connect to a running server using Trenchman is to specify the server URL with the `-s` (`--server`) option. For example, the following command lets you connect to an nREPL server listening on `localhost:12345`:

```sh
$ trench -s nrepl://localhost:12345
```

In addition to nREPL, there is another protocol available: prepl.
To connect to a prepl server, use the `prepl://` scheme instead of `nrepl://`:

```sh
$ trench -s prepl://localhost:5555
```

Also, the connecting port and protocol can be specified with dedicated options:

- port: `-p`, `--port=PORT`
- protocol: `-P`, `--protocol=(nrepl|prepl)`

If you omit the protocol or server host, Trenchman assumes that the following default values are specified:

- protocol: `nrepl`
- server host: `127.0.0.1`

So, in order to connect to `nrepl://127.0.0.1:12345`, you only have to do:

```sh
$ trench -p 12345
```

rather than `trench -s nrepl://127.0.0.1:12345`.

If you omit the port number, Trenchman will read it from a port file, as described in the next section.

#### Port file

A *port file* is a file that contains only the port number that the server is listening on.
Typical nREPL servers generate a port file named `.nrepl-port` at startup.

Trenchman tries to read the port number from a port file if the connecting port is not specified explicitly. By default, Trenchman will read `.nrepl-port` for nREPL connection and `.prepl-port` for prepl connection.

So, the following example connects to `nrepl://127.0.0.1:12345`:

```sh
$ cat .nrepl-port
12345
$ trench
```

If you'd rather use another file as a port file, specify it with the `--port-file` option:

```sh
$ cat my-port-file
3000
$ trench --port-file my-port-file
```

### Evaluation

By default, Trenchman starts a new REPL session after the connection is established:

```sh
$ trench
user=> (println "Hello, World!")
Hello, World!
nil
user=>
```

In addition to starting an ordinary REPL session, Trenchman provides three more
evaluation modes (`-e`/`-f`/`-m`), as with the Clojure CLI.

#### Evaluating an expression (`-e`)

If the `-e` option is specified with an expression, Trenchman evaluates that expression:

```sh
$ trench -e '(println "Hello, World!")'
Hello, World!
$
```

Trenchman will print the evaluation result if the given expression evaluates to a non-`nil` value:

```sh
$ trench -e '(map inc [1 2 3])'
(2 3 4)
$
```

#### Evaluating a file (`-f`)

With the `-f` option, you can load (evaluate) the specified file:

```sh
$ cat hello.clj
(println "Hello, World!")
$ trench -f hello.clj
Hello, World!
$
```

Note that the specified file path is interpreted as one from the client's working directory.
The client will send the entire content of the file to the server once the connection is established.

If `-` is specified as the input file, the input code will be read from stdin:

```sh
$ echo '(println "Hello, World!")' | trench -f-
Hello, World!
$
```

#### Calling `-main` for a namespace (`-m`)

With the `-m` option, you can call the `-main` function for the specified namespace:

```sh
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

Distributed under the MIT License. See LICENSE for more details.
