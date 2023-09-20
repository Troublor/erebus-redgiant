# Combatting Front-Running in Smart Contracts: Attack Mining, Benchmark Construction and Vulnerability Detector Evaluation

- 2023 TSE Publication: https://ieeexplore.ieee.org/document/10108045
- Arxiv version can be found at: https://arxiv.org/abs/2212.12110

This project provides an automated apporach to collect front-running attacks on Ethereum blockchain.
[Front-running](https://medium.com/beaver-smartcontract-security/defi-security-lecture-8-front-running-attack-3247045dd9cd) attack refer to malicious users executed transaction before others to leverage the knowledge for future will-be-executed transactions to make profits.
The profits that can be made by front-running attacks are also known as [Maximal Extractable Value](https://ethereum.org/en/developers/docs/mev/) (MEV) in a sense that miners/validater has the privilege to determine transaction orders, and therefore, control the occurance of front-running attacks.

The collected dataset can be used to analyze front-running attack patterns or evaluate detection tools' capabilities.
Our approach also analyzed the collected attacks to identify contract code that makes front-running attacks possible using dynamic taint analysis.
The vulnerability is previous categorized as Transaction Order Dependency, Event Ordering Bugs, or State Inconsistency Bugs in smart contract.
Our study captures these vulnerabilities from a practical aspect by identify those exploitable vulnerabilities from historical attacks.
The detailed methodology can be found in our paper.

We also leverage our collected dataset, mined from Ethereum mainnet block range `13,000,000 - 13,800,000`, to evaluate seven state-of-the-art smart contract analysis tools to evaluate their front-running vulnerability detection capabilities.

## Project structure

- [`dataset`](./benchmark.md): The dataset of identify historical attacks with vulnerable code located.
- [`tool`](./tool.md): The attack mining tool that can be used to mine more attacks from an Ethereum archive node.
- [`experiment-results`](./experiment-results/README.md): The raw experiment results in the TSE 2023 paper.

## Reusability

To use our code as a library to analyze transactions on Ethereum history, please check our [framework doc](./framework.md).
