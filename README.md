# Sentinel: Simple development server for building web applications

## Features
* Proxy: A configurable proxy server to test your application and its dependencies.
* File Watching: Run commands whenever files change.
* Hot Reload: HTML pages will automatically reload in real time if a watched file change.

## Quick Start

### Install

```bash
go install github.com/flohansen/sentinel/cmd/sentinel@v0.1.0
```

### Initialize config

```bash
sentinel init
```

### Run

```bash
# use the config in the same directory
sentinel
# or specify it by using the config flag
sentinel -config=/path/to/config
```

## How to use the repository

### Setup project
First run

```bash
go generate ./...
```

in the projects root directory. This will generate all files needed to work in
this project.

### Run tests
You run all tests by executing

```bash
go test ./...
```

in the projects root directory.
