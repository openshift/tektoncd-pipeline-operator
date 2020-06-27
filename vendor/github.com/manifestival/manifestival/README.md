# Manifestival

[![Build Status](https://travis-ci.org/manifestival/manifestival.svg?branch=master)](https://travis-ci.org/manifestival/manifestival)

Manipulate unstructured Kubernetes resources loaded from a manifest

## Usage

This library isn't much use without a `Client` implementation. You
have two choices:

- [client-go](https://github.com/manifestival/client-go-client)
- [controller-runtime](https://github.com/manifestival/controller-runtime-client)

## Development

    dep ensure -v
    go test -v ./...
