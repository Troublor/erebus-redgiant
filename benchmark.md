# Dataset of Historical Front-Running Attacks

The dataset `dataset.json.gz` can be downloaded from [Google Drive](https://drive.google.com/file/d/1Cjj0tArosCuBsZp9NrF7A3HvTOmbEnBb/view?usp=sharing).
File `dataset.json.gz` contains the dataset $\mathbb{D}^A$.
One needs to decompress it using `gzip`.
```bash
gzip -d -k dataset.json.gz
```

File `datast.json` will be generated after compression.
It is the entire dataset of attacks we collected as mentioned in the paper.
The JSON file contains an array of front-running attacks.
Each attack $A = \langle T_a, T_v, T_a^p \rangle$ is represented as an JSON object, which contains the following fields:
- `_id`: id of this attack.
- `hash`: the keccak256 hash of the transaction hash of $T_a$, $T_v$ (and $T_a^p$).
- `block`: the block height at which the attack is launched, i.e., the block containing $T_a$.
- `attacker`: the address of the attacker.
- `victim`: the address of the victim.
- `attackTx`: the transaction hash of $T_a$.
- `victimTx`: the transaction hash of $T_v$.
- `profitTx`: the transaction hash of $T_a^p$, if it exists.
- `attackerProfits`: the profits of digital assets obtained by the attacker in attack and attack-free scenarios, respectively.
- `victimProfits`: the profits of digital assets obtained by the victim in attack and attack-free scenarios, respectively.
- `outOfGas` whether this is a gas estimation griefing attack.
- `analysis`: the vulnerability localization analysis results.

The `analysis` field of each attack is an array of influence traces.
Each influence trace is a JSON object containing the following fields:
- `_id`: id of this influence trace.
- `hash`: the keccak256 hash of the shared variable and the hash of the belonging attack.
- `sharedVariable`: the variable in the smart contract that loads the attack altered data in $T_v$ in the attack scenario. In other words, this is the taint source used in dynamic taint analysis.
- `addressingPath`: computations to calculate the address of the variable in contract storage if the `sharedVariable` is a contract storage variable.
- `originalValue`: the value of the `sharedVariable` in the attack-free scenario.
- `alteredValue`: the value of the `sharedVariable` in the attack scenario.
- `writePoint`: the program location where $T_a$ modifies `sharedVariable` in the attack scenario.
- `readPoint`: the program location where $T_v$ loads `sharedVariable` in the attack scenario.
- `consequencePoint`: the program location that directly affects the profits of the victim. In other words, this is the taint sink of dynamic taint analysis.
- `influenceTrace`: the taint flow trace from `readPoint` to `consequencePoint`, i.e., influence trace, in the form of a sequence of contract function invocations.
- `attack`: the id of the attack from which this influence trace is identified.
- `influenceString`: the string representation of `influenceTrace` in the form of a sequence of `${contract address}:${function id}:`
- `influenceString1`: the string representation of `influenceTrace` in the form of a sequence of `${contract code hash}:${function id}:`. This is used to identify duplicate influence traces.

# Benchmark of Attacks with Vulnerability Localized

The benchmark `benchmark.tar.gz` can be downloaded from [Google Drive](https://drive.google.com/file/d/14nd-7PROYsz4QRFswwU-_szgtvSnaS6X/view?usp=sharing).
File `benchmark.tar.gz` contains the benchmark $\mathbb{B}^A$. One needs to decompress it using `tar`:
```bash
tar -xvcf benchmark.tar.gz
```
After decompression, the folder `benchmark` contains two subfolders.
- `attacks`: This folder contains all attacks included in the benchmark. Each attack is represented as a JSON file.
  - Since we focus those attacks with exactly one influence trace, each attack is represented as a influence trace. The file for each attack is named as `${id of influence trace}.attack.json`, and contains the fields of this influence trace. There are also additional fields in the JSON file.
    - `attack`: The attack that each influence trace belongs to.
    - `decodedInfluence`: similar the `influenceTrace` field of the influence trace, but inputs and outputs of each function invocation are decoded.
    - `contractMetas`: Vulnerable contracts functions identified from this influence trace. This fields contains multiple vulnerable contracts that are involved in the influence traces and their related vulnerable functions. The source code is available in `contracts` folder, with relative path specified by `relativePath` field.
- `contracts`: This folder contains all contracts that are referenced by the `contractMetas` fields of attacks in `attacks` folder. Each contract is a folder, which is structured as a [Hardhat](https://hardhat.org) project.
  - Each contract has already been compiled and flattened.
  - The contract runtime bytecode is in `deployedBytecode.bin` file, which is analyzed by tools that analyze bytecode.
  - The flattened source code is in `flattened.sol` file, which is analyzed by tools that analyze source code.
