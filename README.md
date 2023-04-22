# **🫣 I am working on refurnituring this repo by adding source code and giving a detailed description of the benchmark and benchmark collection tool.**
**The current content of this repo is the artifacts for peer review of the TSE publication.**

Preprint can be found at: https://arxiv.org/abs/2212.12110

# Erebus-Redgiant: Implementation of Attack Search and Vulnerability Localization

This repository offers `erebus`, which is the implementation of the search algorithm for historical front-running attacks in Ethereum history and vulnerability localization technique from each attack by extracting influence traces.

`erebus` searches for front-running attacks in the specified block range and save attacks to MongoDB database.
For each found attack, `erebus` will analyze the exploited vulnerability, by extracting influence traces from the attack.
Details can be found in our paper.

## Only Executable before Acceptance

The executable binary of `erebus` can be downloaded from the [Release](https://github.com/erebus-redgiant/tool/releases) page.

We only provide our tool as executable binary files before our paper is accepted.
There are several reasons.
First, the source code of our implementation has some dependencies that will reveal the authors' identities.
Second, we would like to avoid unauthorized use of our code before publication.

We will open source our erebus after the paper is accepted.

## Usage

### Prerequisites

#### Linux

Currently, `erebus` only works on Linux.
We have issues compiling `erebus` on MacOS and Windows.
We are working on fixing these issues.

#### Fully Synced Ethereum Node using Erigon

In order to search for historical front-running attacks, you need to have a [fully synced Ethereum node](https://ethereum.org/en/developers/docs/nodes-and-clients/#full-node) available.
`erebus` requires to use [Erigon](https://github.com/ledgerwatch/erigon) Ethereum client, which is fast to sync from scratch and the disk space consumption is minimal among all available Ethereum clients.
`erebus` directly reads the blockchain database of Erigon on the filesystem to boost searching process.
Please refer to documentation of Erigon for instructions to sync Ethereum blockchain.

#### MongoDB

`erebus` will save searched historical front-running attacks, as well as the vulnerability localization results in MongoDB.

## Running to Collect Attacks and Analyze Vulnerabilities

The executable binary of `erebus` can be downloaded from the [Release](https://github.com/erebus-redgiant/tool/releases) page.

The help information can be found with command:
```bash
erebus collect --help
```

```
Usage:
  erebus collect [flags]

Flags:
  -f, --from uint                      block number to search from (default 64)
  -h, --help                           help for collect
      --localize-timeout string        localize timeout (default "15s")
      --prefetch int                   prefetch degree (default 1)
  -s, --step int                       step size (default 1)
  -t, --to uint                        block number to search to
  -w, --window int                     window size (default 1)
  -p, --window-parallel int            window parallel degree (default 1)
      --window-search-timeout string   window search timeout (default "15s")

Global Flags:
      --concurrency int          (default 1)
      --erigon.datadir string    (default "blockchain/erigon")
      --erigon.rpc string        (default "localhost:9090")
      --eth.url string           (default "http://localhost:8545")
      --log.file string          (default "stdout")
      --log.level uint8          (default 1)
      --log.location
      --log.trace string         (default "logs/trace.log")
      --mongo.database string    (default "erebus")
      --mongo.url string         (default "mongodb://localhost:27017")
      --troubeth.url string      (default "http://localhost:2425")
```

Before running `erebus`, you will need to setup a YAML config file.
An example of the YAML config file is as follows:
```yaml
log:
  level: 1
eth:
  rpc: "ws://localhost:8545"
erigon:
  rpc: "localhost:9090"
  datadir: "/path/to/erigon/database/of/Ethereum/blockchain"
mongo:
  url: "mongodb://localhost:9017"
  database: "erebus"
concurrency: 32
```
`erebus` will automatically use the `config.yaml` file in the current working directory.
You can also setup `EREBUS_CONFIG` environment variable to specify the path to the config file manually.

In the configuration:
- `eth.rpc` is the Ethereum JSON-RPC endpoint URL, which is provided by the Erigon client as described [here](https://github.com/ledgerwatch/erigon#json-rpc-daemon).
- `erigon.rpc` is the Erigon private RPC server address, as described [here](https://github.com/ledgerwatch/erigon/blob/devel/cmd/rpcdaemon/README.md)
- `erigon.datadir` is the path to the blockchain database synced by Erigon (`--datadir` option of Erigon).
- `mongo.url` specifies the URL to connect to the MongoDB.
- `mongo.database` specifies the name of the DB in MongoDB where `erebus` stores attacks in.
- `concurrency` specifies the number of transactions to execute in parallel during searching.

All these configurations can be overridden with command line arguments with corresponding flags.
In addition, there are some other command line flags:
- `--from` and `--to` specifies the range of blocks in which to search for attacks.
- `--window` specifies the size of the block window during searching. Block window consists of several consecutive blocks.
- `--step` specifies the step `erebus` slide the search window.
- `--window-parallel` specifies the number of block windows to search in parallel.
- `--window-search-timeout` specifies the timeout for attack search in each block window.
- `--localize-timeout` specifies the timeout of vulnerability localization for each attack.
- `--prefetch` specifies the number of blocks that that `erebus` will fetch in advance before the block is searched. This flag is meant to optimize the cache hit ratio in disk I/O.

When the attack search and vulnerability localization finished,
you will find the MongoDB is filled with a bunch of attacks in the same structure as described in our [benchmark](https://github.com/erebus-redgiant/benchmark) repository.

## Supplementary Materials of TSE Submission

### Paper Manuscript: Combatting Front-Running in Smart Contracts: Attack Mining, Benchmark Construction and Vulnerability Detectors Evaluation

This GitHub repo holds the following supplementary materials:

- [`benchmark`](./benchmark.md): The dataset of historical front-running attacks we collected and the benchmark of front-running vulnerabilities we constructed.
- [`evaluation-results`](./experiment-results/README.md): The evaluation results of our attack search algorithm, our vulnerability localization approach, and existing vulnerability detection techniques.
