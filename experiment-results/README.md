# Evaluation Results

This repository contains data of the evaluations in our paper, including the evaluation of the attack mining algorithm, the evaluation of the vulnerability localization approach, and the evaluation of state-of-the-art vulnerability detection tools.

## Evaluation of Attack Mining Algorithm

### RQ1-1:

- [Manual Check.xlsx](./Manual%20Check.xlsx) contains the manual check results among three authors to check whether the attacks found by our mining algorithm are true front-running attacks or not. The manual check is conducted on a sample of attacks $\mathbb{D}^S$.

### RQ1-2:

- [displacement.evaluate.output.txt](./displacement.evaluate.output.txt), [insertion.evaluate.output.txt](./insertion.evaluate.output.txt) give the results that our mining algorithm tries to identify displacement attacks and insertion attacks in the baseline dataset.
- The baseline dataset can be found at [Frontrunner-Jones](https://github.com/christoftorres/Frontrunner-Jones).
- There are three columns in the output file. The first column is the attack id in the baseline dataset. The second column marks whether our attack model can identify each attack. The third column marks whether our vulnerability localization approach can localize vulnerable code from the attack.
- [baseline_attack_missing_check.txt](./baseline_attack_missing_check.txt) gives our manual investigation result on the reasons why some attacks are missed by our attack model (sampling with 95% confidence level and 5% confidence interval).

### RQ1-3:

- [rq1-3_search.json](./rq1-3_search.json) shows the attack that our mining algorithm finds in the experiments for RQ1-3.
- The baseline dataset can be found at [Frontrunner-Jones](https://github.com/christoftorres/Frontrunner-Jones).

## Evaluation of Vulnerability Localization

- [Manual Check.xlsx](./Manual%20Check.xlsx) also contains the manual check results among three authors to check whether the underlying vulnerable contract logic is covered by extracted influence trace or not. The manual check is conducted on a sample of attacks $\mathbb{D}^S$.
- [reduction.dat](./reduction.dat) contains the total number of EVM instructions that are marked vulnerable by the naive approach (baseline) and by our approach in each attack of $\mathbb{D}^S$. Used in Fig.3.
- [topNContracts.codehash.csv](./topNContracts.codehash.csv) contains the code hashes of top-1200 contracts used to construct dataset $\mathbb{D}^P$. One can find the corresponding contract from its code hash from dataset $\mathbb{D}^A$, [there](https://github.com/erebus-redgiant/benchmark).
- [saturation.dat](./saturation.dat) contains the number of functions labeled as vulnerable by our approach when increasing the sampling percentage of dataset $\mathbb{D}^P$.

## Evaluation of Vulnerability Detection Tools

Analysis results of each tool on each contract of [dataset] $\mathbb{D}^A$(https://github.com/erebus-redgiant/benchmark) can be downloaded from [Google Drive](https://drive.google.com/file/d/1QhvUmNzB9b2TRwkdHt6A_RBK3ZlvYGMG/view?usp=sharing) (since this is super huge.)
The decompressed folder has the same structure as the benchmark, as described [here](https://github.com/erebus-redgiant/benchmark).
The analysis result of each tool can be found in the `analysis` folder of each contract.
Please refer to the documentation of each tool to learn how to interpret their detection results.
