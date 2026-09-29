# MCR 700 Exam #1 - Prep Key (Exam Info Sheet, Problems 1-5)

The work you should have prepared ahead of time, computed with numpy/scipy. Use it alongside the exam key.

## Problem 1 - BMI (n = 19)

**Summary statistics**

| Statistic | Value | How |
|---|---|---|
| n | 19 | |
| Mean | 27.568 | 523.8/19 |
| Median | 26.2 | 10th ordered value |
| Q1, Q3 | 24.4, 31.4 | values given on exam (software gives 24.5, 30.25 with a different method) |
| IQR | 7.0 | 31.4 - 24.4 |
| Range | 13.9 | 36.4 - 22.5 |
| SS | 293.56 | 14733.9 - 523.8^2/19 |
| Variance | 16.309 | SS/(n-1) |
| Std dev | 4.038 | sqrt(16.309) |
| Skewness (software) | +0.71 | positive |

**Frequency table**

| Class | Count | Rel. freq | Cum. freq | Cum. rel. |
|---|---|---|---|---|
| 20.0 - 24.9 | 8 | 0.421 | 8 | 0.421 |
| 25.0 - 29.9 | 6 | 0.316 | 14 | 0.737 |
| 30.0 - 34.9 | 4 | 0.211 | 18 | 0.947 |
| 35.0 - 39.9 | 1 | 0.053 | 19 | 1.000 |

**Graphs to have ready:** histogram using the four classes (tallest bar first, tapering right); box plot with median 26.2, box 24.4 to 31.4, whiskers to 22.5 and 36.4 (no outliers: upper fence 31.4 + 1.5(7) = 41.9).

**Shape conclusion:** positively (right) skewed - mean > median, histogram tails right, Q3 is farther from the median than Q1.

## Problem 2 - Binomial(n = 20, p = 0.25)

E(X) = 5, SD = sqrt(20(.25)(.75)) = 1.94.

| x | P(X = x) | P(X <= x) |
|---|---|---|
| 0 | .0032 | .0032 |
| 1 | .0211 | .0243 |
| 2 | .0669 | .0913 |
| 3 | .1339 | .2252 |
| 4 | .1897 | .4148 |
| 5 | .2023 | .6172 |
| 6 | .1686 | .7858 |
| 7 | .1124 | .8982 |
| 8 | .0609 | .9591 |
| 9 | .0271 | .9861 |
| 10 | .0099 | .9961 |
| 11 | .0030 | .9991 |

**Decision rules (alpha = .05):**
- Unusually large: P(X >= x) < .05. P(X >= 9) = .0409 (unusual); P(X >= 8) = .1018 (not). So 9 or more is unusual.
- Unusually small: P(X <= x) < .05. P(X <= 1) = .0243 (unusual); P(X <= 2) = .0913 (not). So 1 or fewer is unusual.
- Upper tail formula: P(X >= x) = 1 - P(X <= x-1). Example: P(X >= 10) = 1 - .9861 = .0139.

## Problem 3 - Screening test

|  | D (breast cancer) | D' | Total |
|---|---|---|---|
| Screen + | 68 | 4 | 72 |
| Screen - | 12 | 121 | 133 |
| Total | 80 | 125 | 205 |

**Parameters**
- Se = 68/80 = 0.850
- Sp = 121/125 = 0.968
- Sample PPV = 68/72 = 0.944 and NPV = 121/133 = 0.910 (valid only if the sample prevalence, 80/205 = 39%, is realistic; it is not, so use the tree)

**Tree with prevalence .12**

| Branch | Probability |
|---|---|
| D and + (true positive) | .12 x .85 = .1020 |
| D and - (false negative) | .12 x .15 = .0180 |
| D' and + (false positive) | .88 x .032 = .02816 |
| D' and - (true negative) | .88 x .968 = .85184 |

- P(+) = .13016; P(-) = .86984
- PPV = .1020/.13016 = 0.784
- NPV = .85184/.86984 = 0.979
- P(correct) = .1020 + .85184 = 0.954
- P(error) = .0180 + .02816 = 0.046

**Interpretations**
- Se: among people with breast cancer, 85% screen positive.
- Sp: among people without breast cancer, 96.8% screen negative.
- PPV: among those who screen positive, 78.4% have breast cancer (at 12% prevalence).
- NPV: among those who screen negative, 97.9% do not have breast cancer.
- PPV rises with higher prevalence, Se, or Sp.

## Problem 4 - Case-control (never smokers are the referent)

Counts (yes, no): current 45/77, former 26/69, never 39/137.

| Comparison | OR | 95% CI (printout) | Contains 1? | Conclusion |
|---|---|---|---|---|
| Current vs never | 45(137)/(77(39)) = **2.05** | (1.2307, 3.4244) | No | Significant; higher odds of lung cancer |
| Former vs never | 26(137)/(69(39)) = **1.32** | (0.7453, 2.3510) | Yes | Not significant |
| Ever vs never (collapsed) | 71(137)/(146(39)) = **1.71** | (1.08, 2.69) (my Woolf calc) | No | Significant |

- CI check (Woolf): ln(2.053) +/- 1.96 sqrt(1/45 + 1/77 + 1/39 + 1/137) reproduces the printout limits.
- Relative risk lines on the printout (Column 1/2) are **not valid** for case-control data, because the row totals reflect sampling, not risk. Only the OR is a valid measure. It approximates the RR when the disease is rare.
- Estimable: P(exposure given disease status), e.g. P(current smoker given lung cancer) = 45/110 = .409. Not estimable: P(disease given exposure) (needs a cohort or cross-sectional design).

## Problem 5 - Reaction time

| Analysis | Mean | SD | SE | 95% CI | t | df | p |
|---|---|---|---|---|---|---|---|
| No training vs 400 | 406.3 | 45.72 | 6.279 | (393.7, 418.9) | 1.00 | 52 | .3217 |
| Training vs 400 | 382.2 | 27.17 | 4.351 | (373.4, 391.0) | -4.10 | 38 | .0002 |

**Critique and interpretation**
- No training: CI contains 400 and p > .05, so no evidence mean differs from 400.
- Training: CI excludes 400 and p < .05, so mean differs from 400 (it is lower).
- Two groups: independent samples. Difference 24.1 (no training minus trained).
- Equality of variances: folded F = 2.83, p = .0011, so variances differ. Use Satterthwaite: t = 3.16, df = 86.6, p = .0022, CI (8.92, 39.29). (Pooled, not appropriate: t = 2.93, df 90, p = .0043, CI (7.77, 40.44).)
- Conclusion: mean reaction time differs; trained women are faster by roughly 9 to 39 units.
- Tests were two-sided. A "trained are faster" claim would be one-sided (Ha: mu_trained < mu_no_train) with p about .0011.

**Different null value (likely exam question)**: t = (406.3 - mu0)/6.279, df = 52.

| mu0 | t | two-sided p |
|---|---|---|
| 400 | 1.00 | .3217 |
| 410 | -0.59 | .558 |
| 420 | -2.18 | about .034 (t table: between .02 and .05 at df 52; critical value 2.007, so reject at .05) |

Note 420 lies outside the 95% CI (393.7, 418.9), consistent with rejection.
