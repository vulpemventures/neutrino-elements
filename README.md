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
neutrinod also exposes HTTP endpoint that supports adding webhooks to receive notifications when an event occurs.<br>
- `neutrino` is a simple command line tool that can be used to watch Liquid side-chain events.<br>

## Local Development

### Requirements
- Go (^1.17.*)
- Nigiri (^0.4.*)

### Build neutrinod & neutrino CLI

```
make build-nd
make build-n
```

### Run test

```
make test
```

## Usage

### Run neutrinod

```
./neutrinod
```

### Config CLI
```
./neutrino config
```

### Watch for events related to wallet-descriptor(websocket-{HOST:PORT}/neutrino/subscribe/ws) 
```
./neutrino subscribe --descriptor="{WALLET_DESCRIPTOR}" --block_height={BLOCK_HEIGHT} --events=unspentUtxo
```
### Add webhook(http)
```sh
curl -X POST http://localhost:8000/neutrino/subscribe/http \
   -H 'Content-Type: application/json' \
   -d '{"actionType":"register", "eventTypes":["unspentUtxo"], "descriptorWallet":"wpkh(037470e26cc774eca62ca19e1a182461a5f3d3680acbc593ce3f38cd142c26c03d)", "startBlockHeight":0, "endpointUrl":"http://127.0.0.1:62900"}'
```

## API
Neutrinod can be used to subscribe to events related to wallet-descriptor using web-socket or by registrating webhook using HTTP.<br>

### Web-socket
To subscribe to events related to wallet-descriptor using web-socket connection, send below json :<br>
```json
{
    "actionType":"register", 
    "eventTypes":["unspentUtxo"], 
    "descriptorWallet":"{WALLET_DESCRIPTOR}", 
    "startBlockHeight":"{START_BLOCK_HEIGHT}", 
}
```
to the following endpoint:
```
/neutrino/subscribe/ws
```

### HTTP 
To register webhook, send bellow json :<br>
```json
{
  "actionType": "register",
  "eventTypes": ["unspentUtxo"],
  "descriptorWallet": "{WALLET_DESCRIPTOR}",
  "startBlockHeight": "{START_BLOCK_HEIGHT}",
  "endpointUrl": "{ENDPOINT_URL}"
}
```
to the following endpoint:
```
/neutrino/subscribe/http
```

Valid actionTypes: "register", "unregister"<br>
Valid eventTypes: "unspentUtxo", "spentUtxo"<br>

## License

MIT - see the LICENSE.md file for details

## Acknowledgments

* [Neutrino - Light bitcoin client](https://github.com/lightninglabs/neutrino)
* [Compact Block Filters for Light Clients - BIP158](https://github.com/bitcoin/bips/blob/master/bip-0158.mediawiki)
* [tinybit](https://github.com/Jeiwan/tinybit)
