# Sentinel: Simple development server for building web applications
![release](https://github.com/flohansen/sentinel/actions/workflows/release.yaml/badge.svg)
![version](https://img.shields.io/github/v/release/flohansen/sentinel)

## Features
* Proxy: A configurable proxy server to test your application and its dependencies.
* File Watching: Run commands whenever files change.
* Hot Reload: HTML pages will automatically reload in real time if a watched file change.

## Quick Start

### Install

```bash
export OS=$(uname | awk '{print tolower($0)}')
export ARCH=$(case $(uname -m) in x86_64) echo -n amd64 ;; aarch64) echo -n arm64 ;; *) echo -n $(uname -m) ;; esac)
wget "https://github.com/flohansen/sentinel/releases/download/v0.4.2/sentinel_${OS}_${ARCH}.tar.gz"
sudo tar -xzf "sentinel_${OS}_${ARCH}.tar.gz" -C /usr/local/bin
```

### Initialize config

```bash
sentinel init
```

### Run

```bash
# use the config in the same directory
sentinel run
# or specify it by using the config flag
sentinel run -config=/path/to/config
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
