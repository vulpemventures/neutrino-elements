# Neutrino Elements

[![Go](https://github.com/vulpemventures/neutrino-elements/actions/workflows/ci.yml/badge.svg)](https://github.com/vulpemventures/neutrino-elements/actions/workflows/ci.yml)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/vulpemventures/neutrino-elements)](https://pkg.go.dev/github.com/vulpemventures/neutrino-elements)
[![Release](https://img.shields.io/github/release/vulpemventures/neutrino-elements.svg?style=flat-square)](https://github.com/vulpemventures/neutrino-elements/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/vulpemventures/neutrino-elements)](https://goreportcard.com/report/github.com/vulpemventures/neutrino-elements)
[![Bitcoin Donate](https://badgen.net/badge/Bitcoin/Donate/F7931A?icon=bitcoin)](https://blockstream.info/address/3MdERN32qiMnQ68bSSee5CXQkrSGx1iStr)

Neutrino + Elements

## Overview

Neutrino-elements is a set of useful packages and binaries that can be used to "watch" Liquid side-chain events.<br>
It uses Compact Block Filter (BIP0158) to implement a light client for [elements](https://elementsproject.org/) based networks.<br>

Two packages, that works independently, are provided if you want to build your own light client:<br>
- `NodeService` is a full node maintaining an up-to-date state of the block headers + compact filters. The NodeService writes down headers and filters in repositories.<br>
- `ScannerService` uses filters and headers repositories to handle `ScanRequest` which aims to know if an outpoint (identified by its script) is spent or not.<br>

Two binaries are provided if you want to use ready light client:<br>
- `neutrinod` is a daemon that accepts websocket connections on which clients can send requests to watch for events related to wallet-descriptor<br>
- `neutrino` is a simple command line tool that can be used to watch Liquid side-chain events.<br>

## Getting Started with neutrinod

### Build neutrinod & neutrino CLI

```
make build-n
make build-nd
```

### Start neutrinod

```
./bin/neutrinod
```

### Use neutrino CLI

#### Config CLI
```
./bin/neutrino config
```

#### Watch for events related to wallet-descriptor
```
./bin/neutrino subscribe --descriptor="{WALLET_DESCRIPTOR}" --block_height={BLOCK_HEIGHT}
```

## License

MIT - see the LICENSE.md file for details

## Acknowledgments

* [Neutrino - Light bitcoin client](https://github.com/lightninglabs/neutrino)
* [Compact Block Filters for Light Clients - BIP158](https://github.com/bitcoin/bips/blob/master/bip-0158.mediawiki)
* [tinybit](https://github.com/Jeiwan/tinybit)
