# Neutrino Elements

Neutrino + Elements

## Description

neutrino-elements uses Compact Block Filter (BIP0158) to implement a light client for [elements](https://elementsproject.org/)-based networks.

Two services are provided, they can work independantly:
- `NodeService` is a full node maintaining an up-to-date state of the block headers + compact filters. The NodeService writes down headers and filters in repositories.
- `UtxoScanner` uses filters and headers repositories to handle `ScanRequest` which aims to know if an outpoint (identified by its script) is spent or not. 

## Getting Started

### Build

```
make build
```

### Unit tests

```
make test
```

### Format

```
make fmt
```

## License

MIT - see the LICENSE.md file for details

## Acknowledgments

* [Neutrino - Light bitcoin client](https://github.com/lightninglabs/neutrino)
* [Compact Block Filters for Light Clients - BIP158](https://github.com/bitcoin/bips/blob/master/bip-0158.mediawiki)
* [tinybit](https://github.com/Jeiwan/tinybit)
