# Sentinel: Simplistic development server for building web applications

## Features
* Proxy: A configurable proxy server to test your application and its dependencies.
* File Watching: Run commands whenever files change.
* Hot Reload: HTML pages will automatically reload in real time if a watched file change.

## Contribute

### Setup project
First run

    go generate ./...

in the projects root directory. This will generate all files needed to work in
this project.

### Run tests
You run all tests by executing

    go test ./...

in the projects root directory.
