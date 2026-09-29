# MCR 700 / Biostats Exam #1 - Master Answer Key

Part A is the exam key. Part B is the prep work from the exam info sheet.

---

# PART A - EXAM ANSWER KEY


All numbers were computed in Python (numpy/scipy) and checked against the SAS printouts.

## A. Problem 1 - BMI (n = 19)

Tools: sum = 523.8, sum of squares = 14733.9.

| Part | Answer | Reasoning |
|---|---|---|
| i | Mean = **27.57** | 523.8 / 19 = 27.568 |
| ii | Median = **26.2** | n odd, so the 10th ordered value |
| iii | **c) 4** | SS = 14733.9 - 523.8^2/19 = 293.56; var = 293.56/18 = 16.31; s = 4.04 |
| iv | **c) positively skewed** | Mean (27.57) > median (26.2); Q3 - median = 5.2 vs median - Q1 = 1.8 |
| v | Mean 27.57 > median 26.2, and upper quartile is farther from median (5.2) than lower quartile (1.8) | Long right tail (max 36.4) pulls mean up |
| vi | **6** | 26.1, 26.2, 27.5, 28.6, 28.9, 29.1 |
| vii | P(30.0 < X < 34.9) = 4/19 = **0.211** | 31.4, 31.5, 32.4, 34.1 |
| viii | P(X < 30.0) = 14/19 = **0.737** | 8 values in 20.0-24.9 plus 6 in 25.0-29.9 |

## A. Problem 2 - Flu shot adverse reaction, n = 20, p = 0.25

| Part | Answer | Reasoning |
|---|---|---|
| i | **5** | E = np = 20(.25) |
| ii | P(X >= 10) = 1 - P(X <= 9) = 1 - 0.9861 = **0.0139** | Binomial(20, .25); "unusually large" is a upper-tail probability |
| iii | **Yes** | 0.0139 < 0.05 |
| iv | The **p-value** | Probability of a result at least this extreme if H0 (p = .25) is true |
| v | **c) proportion is greater than .25** | Reject H0 in the upper tail |
| vi | P(X <= 3) = **0.2252**; **not** unusually small | Lower tail; 0.2252 > 0.05 |

## A. Problem 3 - Breast cancer screen (80 with disease, 125 without)

| Part | Answer | Reasoning |
|---|---|---|
| i | Se = 68/80 = **0.85**; Sp = 121/125 = **0.968** | Negatives without disease = 125 - 4 = 121 |
| ii | Among people who have breast cancer, 85% test positive | Se = P(+ given D) |
| iii | PPV = .102 / (.102 + .02816) = **0.784** | Tree, prevalence .12: TP = .12(.85) = .102; FP = .88(.032) = .02816 |
| iv | Of people who screen positive (in a population with 12% prevalence), about 78.4% actually have breast cancer | PPV = P(D given +) |
| v | **b)** | NPV = of those who test negative, proportion without disease |
| vi | False positive = .88(.032) = **0.0282** | D' then + |
| vii | Negative screen = .12(.15) + .88(.968) = .018 + .85184 = **0.8698** | Both negative branches |
| viii | Correct = .102 + .85184 = **0.9538** | TP + TN |

## A. Problem 4 - Case-control, smoking and lung cancer

| Part | Answer | Reasoning |
|---|---|---|
| i | OR = (45)(137)/((77)(39)) = **2.05** | Current vs never |
| ii | Current smokers are estimated to have about **2.05 times the risk** (odds) of lung cancer compared with never smokers | OR approximates RR only when disease is rare |
| iii | **(1.2307, 3.4244)** | From printout |
| iv | **Yes** | CI excludes 1 |
| v | The 95% CI for the OR (1.23 to 3.42) does not contain 1 | So OR is significantly > 1 |
| vi | OR = (26)(137)/((69)(39)) = **1.32** | Former vs never |
| vii | **No** | CI (0.7453, 2.3510) contains 1 |
| viii | OR = (71)(137)/((146)(39)) = **1.71** | Ever vs never |
| ix | a) **No**; b) **Yes** | Case-control fixes the number of cases/controls, so only P(exposure given disease status) is estimable; P(disease given exposure) is not |

## A. Problem 5 - Reaction time

| Part | Answer | Reasoning |
|---|---|---|
| i | **(393.7, 418.9)** | 406.3 +/- 2.007(6.279); t(52, .975) = 2.007 |
| ii | We are 95% confident the true mean reaction time for untrained women is between 393.7 and 418.9 | |
| iii | H0: mu = 400 vs Ha: mu != 400 | |
| iv | p = **0.3217**; fail to reject H0; no evidence the mean differs from 400 | t = (406.3 - 400)/6.279 = 1.00, df 52 |
| v | p = **0.0002**; reject H0; evidence the trained mean differs from 400 (it is lower, 382.2) | t = -4.10, df 38 |
| vi | The 95% CI (373.4, 391.0) does not contain 400, so reject H0 at alpha = .05 | CI and two-sided test agree |
| vii | **Independent** | Different women in each group |
| viii | **24.1** (no training minus trained) | 406.3 - 382.2 |
| ix | **No** | s = 45.7 vs 27.2; folded F = 2.83, p = 0.0011 |
| x | **0.0011** | Folded F test for equal variances |
| xi | p = **0.0022** (Satterthwaite, unequal variances); reject H0; evidence the mean reaction times differ | t = 3.16, df 86.6 |
| xii | **No** | The test done was two-sided |
| xiii | H0: mu_trained >= mu_no_train vs Ha: mu_trained < mu_no_train (equivalently H0: mu_no - mu_tr <= 0 vs Ha: > 0) | Direction was specified in advance; one-sided p would be about 0.0011 |

## A. Problem 6 - Systolic BP box plots (begin vs follow-up)

i) The three concepts are center, spread, and shape.
- a) **Center:** follow-up is lower; median about 134 vs about 144 at the beginning.
- b) **Spread:** follow-up is much less variable; IQR about 8 (130-138) vs about 16 (136-152); range about 36 vs about 78.
- c) **Shape:** both roughly symmetric (mean close to median in each; whiskers about equal on each side); begin has a slightly longer upper whisker.

ii) **b) paired t-test** - same people measured twice.

## A. Problem 7 - Percent body fat, HD ~ N(25, 5), No HD ~ N(12, 4)

| Part | Answer | Reasoning |
|---|---|---|
| i | **Large** | Disease group has the higher mean |
| ii | Se = P(X > 15 given HD) = P(Z > (15-25)/5 = -2) = **0.9772** | |
| iii | **b)** | Raising the cutoff for "large = positive" lowers sensitivity (.919) and raises specificity (.933) |
| iv | **ROC curve** | |
| v | **c) both** | Closest to (0,1) corner, or farthest from diagonal (Youden) |

Extra check for iii: at cutoff 15, Sp = P(Z < .75) = .773; at 18, Sp = P(Z < 1.5) = .933.

## A. Problem 8 - Multiple choice

| Part | Answer | Reasoning |
|---|---|---|
| i | **c** numeric, discrete | A count |
| ii | **a** | Players are a narrow age band; crowd spans all ages |
| iii | **b** median | Others measure spread; median measures center |
| iv | **b** 0 | |
| v | **d** permutations | Order matters |
| vi | **d** all of the above | |
| vii | **b** | Rare disease assumption |
| viii | **a** | CI estimates the population mean |
| ix | **b** | Hypotheses concern parameters, not statistics |
| x | Free point (any answer). Objectively, the answer is c) Tom Brady | |

---

# PART B - EXAM INFO PREP KEY


The work you should have prepared ahead of time, computed with numpy/scipy. Use it alongside the exam key.

## B. Problem 1 - BMI (n = 19)

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

## B. Problem 2 - Binomial(n = 20, p = 0.25)

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

## B. Problem 3 - Screening test

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

## B. Problem 4 - Case-control (never smokers are the referent)

Counts (yes, no): current 45/77, former 26/69, never 39/137.

| Comparison | OR | 95% CI (printout) | Contains 1? | Conclusion |
|---|---|---|---|---|
| Current vs never | 45(137)/(77(39)) = **2.05** | (1.2307, 3.4244) | No | Significant; higher odds of lung cancer |
| Former vs never | 26(137)/(69(39)) = **1.32** | (0.7453, 2.3510) | Yes | Not significant |
| Ever vs never (collapsed) | 71(137)/(146(39)) = **1.71** | (1.08, 2.69) (my Woolf calc) | No | Significant |

- CI check (Woolf): ln(2.053) +/- 1.96 sqrt(1/45 + 1/77 + 1/39 + 1/137) reproduces the printout limits.
- Relative risk lines on the printout (Column 1/2) are **not valid** for case-control data, because the row totals reflect sampling, not risk. Only the OR is a valid measure. It approximates the RR when the disease is rare.
- Estimable: P(exposure given disease status), e.g. P(current smoker given lung cancer) = 45/110 = .409. Not estimable: P(disease given exposure) (needs a cohort or cross-sectional design).

## B. Problem 5 - Reaction time

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
