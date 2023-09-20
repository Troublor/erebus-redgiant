# erebus-redgiant

## Description

erebus-redgiant is a golang framework that allows flexible Ethereum transaction analysis using EVM on an Ethereum Archive Node.
Main functionalities include:
- Fork blockchain on any block height (and transaction index).
- Execute transactions on the forked blockchain.
- Analyze the execution result of the transactions.
  - Taint analysis
  - Structured call trace extraction
  - and more...