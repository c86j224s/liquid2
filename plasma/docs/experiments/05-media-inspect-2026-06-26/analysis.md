# Media Inspect Statistical Analysis

Codex image attachments stood in for future explicit Plasma inspect tools.
M1/M2 are always-inspect variants. M1C/M2C are conditional variants that first read metadata and request inspect only when needed.

## Conclusion

This analysis currently includes 90 completed report runs and 72 judge comparisons.
Judge results are the primary decision signal. Automatic scores and auto hard flags are heuristic support data only.

- C1-M0-vs-M1: right-hand variant won; A=0, B=6, TIE=0, n=6, sign-test p=0.0312.
- C1-M0-vs-M1C: right-hand variant won; A=0, B=6, TIE=0, n=6, sign-test p=0.0312.
- C1-M1-vs-M1C: split or tied; A=3, B=3, TIE=0, n=6, sign-test p=1.0000.
- C2-M0-vs-M2: right-hand variant won; A=0, B=6, TIE=0, n=6, sign-test p=0.0312.
- C2-M0-vs-M2C: split or tied; A=3, B=3, TIE=0, n=6, sign-test p=1.0000.
- C2-M2-vs-M2C: right-hand variant won; A=2, B=4, TIE=0, n=6, sign-test p=0.6875.
- C3-M0-vs-M1: right-hand variant won; A=0, B=6, TIE=0, n=6, sign-test p=0.0312.
- C3-M0-vs-M1C: right-hand variant won; A=0, B=6, TIE=0, n=6, sign-test p=0.0312.
- C3-M1-vs-M1C: split or tied; A=3, B=3, TIE=0, n=6, sign-test p=1.0000.
- C4-M1-vs-M1C: right-hand variant won; A=2, B=4, TIE=0, n=6, sign-test p=0.6875.
- C5-M1-vs-M1C: right-hand variant won; A=0, B=6, TIE=0, n=6, sign-test p=0.0312.
- C6-M1-vs-M1C: right-hand variant won; A=2, B=4, TIE=0, n=6, sign-test p=0.6875.

## Variant Summary

| Mission | Variant | Runs | Mean score | Median score | Mean sources | Mean visual/OCR signals | Inspect requested | Auto hard flags |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| C1 | M0 | 6 | 26.50 | 26.25 | 1.00 | 32.00 | 0/6 | 2 |
| C1 | M1 | 6 | 32.67 | 32.25 | 2.00 | 31.83 | 0/6 | 0 |
| C1 | M1C | 6 | 30.75 | 31.25 | 2.00 | 30.50 | 6/6 | 0 |
| C2 | M0 | 6 | 32.58 | 32.50 | 2.00 | 35.17 | 0/6 | 2 |
| C2 | M2 | 6 | 35.00 | 37.25 | 1.33 | 46.17 | 0/6 | 0 |
| C2 | M2C | 6 | 35.75 | 36.25 | 2.00 | 41.00 | 0/6 | 3 |
| C3 | M0 | 6 | 21.58 | 21.75 | 0.00 | 32.67 | 0/6 | 0 |
| C3 | M1 | 6 | 25.67 | 25.50 | 1.00 | 30.33 | 0/6 | 0 |
| C3 | M1C | 6 | 20.42 | 20.50 | 1.00 | 23.33 | 6/6 | 0 |
| C4 | M1 | 6 | 30.00 | 30.75 | 1.00 | 40.00 | 0/6 | 0 |
| C4 | M1C | 6 | 30.17 | 30.75 | 1.00 | 40.33 | 6/6 | 0 |
| C5 | M1 | 6 | 30.50 | 29.50 | 1.00 | 41.00 | 0/6 | 0 |
| C5 | M1C | 6 | 27.75 | 27.75 | 1.00 | 36.00 | 6/6 | 0 |
| C6 | M1 | 6 | 31.83 | 31.50 | 1.00 | 40.17 | 0/6 | 0 |
| C6 | M1C | 6 | 28.67 | 28.00 | 1.00 | 35.83 | 6/6 | 0 |

## Runs

| Run | Mission | Variant | Status | Words | Sources | Visual/OCR signals | Uncertainty signals | Score | Auto hard flag |
|---|---|---|---:|---:|---:|---:|---:|---:|---|
| C1-M0-seed-0001 | C1 | M0 | completed | 472 | 2 | 35 | 2 | 33.5 | False |
| C1-M0-seed-0002 | C1 | M0 | completed | 498 | 2 | 34 | 2 | 33.0 | True |
| C1-M0-seed-0003 | C1 | M0 | completed | 411 | 2 | 30 | 1 | 29.5 | False |
| C1-M0-seed-0004 | C1 | M0 | completed | 456 | 0 | 30 | 2 | 21.0 | True |
| C1-M0-seed-0005 | C1 | M0 | completed | 461 | 0 | 34 | 2 | 23.0 | False |
| C1-M0-seed-0006 | C1 | M0 | completed | 361 | 0 | 29 | 1 | 19.0 | False |
| C1-M1-seed-0001 | C1 | M1 | completed | 398 | 2 | 29 | 3 | 32.0 | False |
| C1-M1-seed-0002 | C1 | M1 | completed | 399 | 2 | 40 | 4 | 39.0 | False |
| C1-M1-seed-0003 | C1 | M1 | completed | 350 | 2 | 30 | 1 | 29.5 | False |
| C1-M1-seed-0004 | C1 | M1 | completed | 390 | 2 | 29 | 0 | 27.5 | False |
| C1-M1-seed-0005 | C1 | M1 | completed | 415 | 2 | 27 | 4 | 32.5 | False |
| C1-M1-seed-0006 | C1 | M1 | completed | 496 | 2 | 36 | 3 | 35.5 | False |
| C1-M1C-seed-0001 | C1 | M1C | completed | 423 | 2 | 29 | 2 | 30.5 | False |
| C1-M1C-seed-0002 | C1 | M1C | completed | 321 | 2 | 19 | 2 | 25.5 | False |
| C1-M1C-seed-0003 | C1 | M1C | completed | 535 | 2 | 38 | 3 | 36.5 | False |
| C1-M1C-seed-0004 | C1 | M1C | completed | 407 | 2 | 35 | 1 | 32.0 | False |
| C1-M1C-seed-0005 | C1 | M1C | completed | 356 | 2 | 36 | 1 | 32.5 | False |
| C1-M1C-seed-0006 | C1 | M1C | completed | 405 | 2 | 26 | 1 | 27.5 | False |
| C2-M0-seed-0001 | C2 | M0 | completed | 336 | 2 | 33 | 1 | 31.0 | False |
| C2-M0-seed-0002 | C2 | M0 | completed | 443 | 2 | 32 | 2 | 32.0 | True |
| C2-M0-seed-0003 | C2 | M0 | completed | 510 | 2 | 40 | 0 | 33.0 | False |
| C2-M0-seed-0004 | C2 | M0 | completed | 431 | 2 | 32 | 2 | 32.0 | False |
| C2-M0-seed-0005 | C2 | M0 | completed | 399 | 2 | 38 | 1 | 33.5 | False |
| C2-M0-seed-0006 | C2 | M0 | completed | 362 | 2 | 36 | 2 | 34.0 | True |
| C2-M2-seed-0001 | C2 | M2 | completed | 331 | 2 | 47 | 3 | 41.0 | False |
| C2-M2-seed-0002 | C2 | M2 | completed | 380 | 2 | 44 | 1 | 36.5 | False |
| C2-M2-seed-0003 | C2 | M2 | completed | 324 | 0 | 36 | 1 | 22.5 | False |
| C2-M2-seed-0004 | C2 | M2 | completed | 400 | 0 | 53 | 1 | 31.0 | False |
| C2-M2-seed-0005 | C2 | M2 | completed | 341 | 2 | 44 | 2 | 38.0 | False |
| C2-M2-seed-0006 | C2 | M2 | completed | 441 | 2 | 53 | 1 | 41.0 | False |
| C2-M2C-seed-0001 | C2 | M2C | completed | 407 | 2 | 38 | 2 | 35.0 | True |
| C2-M2C-seed-0002 | C2 | M2C | completed | 419 | 2 | 46 | 1 | 37.5 | True |
| C2-M2C-seed-0003 | C2 | M2C | completed | 274 | 2 | 37 | 1 | 33.0 | False |
| C2-M2C-seed-0004 | C2 | M2C | completed | 406 | 2 | 47 | 2 | 39.5 | True |
| C2-M2C-seed-0005 | C2 | M2C | completed | 449 | 2 | 50 | 1 | 39.5 | False |
| C2-M2C-seed-0006 | C2 | M2C | completed | 313 | 2 | 28 | 2 | 30.0 | False |
| C3-M0-seed-0001 | C3 | M0 | completed | 238 | 0 | 25 | 0 | 15.5 | False |
| C3-M0-seed-0002 | C3 | M0 | completed | 281 | 0 | 28 | 1 | 18.5 | False |
| C3-M0-seed-0003 | C3 | M0 | completed | 344 | 0 | 38 | 0 | 22.0 | False |
| C3-M0-seed-0004 | C3 | M0 | completed | 275 | 0 | 30 | 3 | 22.5 | False |
| C3-M0-seed-0005 | C3 | M0 | completed | 293 | 0 | 34 | 1 | 21.5 | False |
| C3-M0-seed-0006 | C3 | M0 | completed | 320 | 0 | 41 | 4 | 29.5 | False |
| C3-M1-seed-0001 | C3 | M1 | completed | 325 | 1 | 25 | 2 | 23.5 | False |
| C3-M1-seed-0002 | C3 | M1 | completed | 372 | 1 | 34 | 1 | 26.5 | False |
| C3-M1-seed-0003 | C3 | M1 | completed | 292 | 1 | 29 | 2 | 25.5 | False |
| C3-M1-seed-0004 | C3 | M1 | completed | 326 | 1 | 26 | 3 | 25.5 | False |
| C3-M1-seed-0005 | C3 | M1 | completed | 280 | 1 | 23 | 1 | 21.0 | False |
| C3-M1-seed-0006 | C3 | M1 | completed | 392 | 1 | 45 | 1 | 32.0 | False |
| C3-M1C-seed-0001 | C3 | M1C | completed | 292 | 1 | 27 | 1 | 23.0 | False |
| C3-M1C-seed-0002 | C3 | M1C | completed | 270 | 1 | 21 | 0 | 18.5 | False |
| C3-M1C-seed-0003 | C3 | M1C | completed | 297 | 1 | 24 | 0 | 20.0 | False |
| C3-M1C-seed-0004 | C3 | M1C | completed | 237 | 1 | 22 | 0 | 19.0 | False |
| C3-M1C-seed-0005 | C3 | M1C | completed | 303 | 1 | 23 | 1 | 21.0 | False |
| C3-M1C-seed-0006 | C3 | M1C | completed | 310 | 1 | 23 | 1 | 21.0 | False |
| C4-M1-seed-0001 | C4 | M1 | completed | 358 | 1 | 44 | 1 | 31.5 | False |
| C4-M1-seed-0002 | C4 | M1 | completed | 270 | 1 | 37 | 1 | 28.0 | False |
| C4-M1-seed-0003 | C4 | M1 | completed | 365 | 1 | 41 | 2 | 31.5 | False |
| C4-M1-seed-0004 | C4 | M1 | completed | 296 | 1 | 44 | 1 | 31.5 | False |
| C4-M1-seed-0005 | C4 | M1 | completed | 300 | 1 | 35 | 3 | 30.0 | False |
| C4-M1-seed-0006 | C4 | M1 | completed | 286 | 1 | 39 | 0 | 27.5 | False |
| C4-M1C-seed-0001 | C4 | M1C | completed | 324 | 1 | 41 | 1 | 30.0 | False |
| C4-M1C-seed-0002 | C4 | M1C | completed | 247 | 1 | 27 | 1 | 23.0 | False |
| C4-M1C-seed-0003 | C4 | M1C | completed | 307 | 1 | 35 | 1 | 27.0 | False |
| C4-M1C-seed-0004 | C4 | M1C | completed | 331 | 1 | 41 | 2 | 31.5 | False |
| C4-M1C-seed-0005 | C4 | M1C | completed | 362 | 1 | 47 | 1 | 33.0 | False |
| C4-M1C-seed-0006 | C4 | M1C | completed | 386 | 1 | 51 | 2 | 36.5 | False |
| C5-M1-seed-0001 | C5 | M1 | completed | 306 | 1 | 43 | 0 | 29.5 | False |
| C5-M1-seed-0002 | C5 | M1 | completed | 295 | 1 | 37 | 2 | 29.5 | False |
| C5-M1-seed-0003 | C5 | M1 | completed | 302 | 1 | 40 | 1 | 29.5 | False |
| C5-M1-seed-0004 | C5 | M1 | completed | 362 | 1 | 45 | 2 | 33.5 | False |
| C5-M1-seed-0005 | C5 | M1 | completed | 313 | 1 | 38 | 1 | 28.5 | False |
| C5-M1-seed-0006 | C5 | M1 | completed | 323 | 1 | 43 | 2 | 32.5 | False |
| C5-M1C-seed-0001 | C5 | M1C | completed | 285 | 1 | 40 | 1 | 29.5 | False |
| C5-M1C-seed-0002 | C5 | M1C | completed | 247 | 1 | 29 | 2 | 25.5 | False |
| C5-M1C-seed-0003 | C5 | M1C | completed | 356 | 1 | 42 | 1 | 30.5 | False |
| C5-M1C-seed-0004 | C5 | M1C | completed | 285 | 1 | 34 | 3 | 29.5 | False |
| C5-M1C-seed-0005 | C5 | M1C | completed | 307 | 1 | 35 | 0 | 25.5 | False |
| C5-M1C-seed-0006 | C5 | M1C | completed | 307 | 1 | 36 | 0 | 26.0 | False |
| C6-M1-seed-0001 | C6 | M1 | completed | 351 | 1 | 44 | 4 | 36.0 | False |
| C6-M1-seed-0002 | C6 | M1 | completed | 384 | 1 | 51 | 3 | 38.0 | False |
| C6-M1-seed-0003 | C6 | M1 | completed | 319 | 1 | 41 | 0 | 28.5 | False |
| C6-M1-seed-0004 | C6 | M1 | completed | 361 | 1 | 35 | 2 | 28.5 | False |
| C6-M1-seed-0005 | C6 | M1 | completed | 295 | 1 | 29 | 2 | 25.5 | False |
| C6-M1-seed-0006 | C6 | M1 | completed | 319 | 1 | 41 | 4 | 34.5 | False |
| C6-M1C-seed-0001 | C6 | M1C | completed | 346 | 1 | 33 | 1 | 26.0 | False |
| C6-M1C-seed-0002 | C6 | M1C | completed | 338 | 1 | 38 | 2 | 30.0 | False |
| C6-M1C-seed-0003 | C6 | M1C | completed | 294 | 1 | 34 | 0 | 25.0 | False |
| C6-M1C-seed-0004 | C6 | M1C | completed | 365 | 1 | 42 | 4 | 35.0 | False |
| C6-M1C-seed-0005 | C6 | M1C | completed | 330 | 1 | 40 | 3 | 32.5 | False |
| C6-M1C-seed-0006 | C6 | M1C | completed | 283 | 1 | 28 | 1 | 23.5 | False |

## Paired Score Comparisons

### C1: M1 minus M0

- paired blocks: 6
- wins: 4/6
- mean diff: 6.17
- median diff: 6.25
- sign-test p-value: 0.3750
- block diffs: 0001:-1.50, 0002:6.00, 0003:0.00, 0004:6.50, 0005:9.50, 0006:16.50

### C1: M1C minus M0

- paired blocks: 6
- wins: 4/6
- mean diff: 4.25
- median diff: 7.75
- sign-test p-value: 0.6875
- block diffs: 0001:-3.00, 0002:-7.50, 0003:7.00, 0004:11.00, 0005:9.50, 0006:8.50

### C1: M1C minus M1

- paired blocks: 6
- wins: 2/6
- mean diff: -1.92
- median diff: -0.75
- sign-test p-value: 1.0000
- block diffs: 0001:-1.50, 0002:-13.50, 0003:7.00, 0004:4.50, 0005:0.00, 0006:-8.00

### C2: M2 minus M0

- paired blocks: 6
- wins: 4/6
- mean diff: 2.42
- median diff: 4.50
- sign-test p-value: 0.6875
- block diffs: 0001:10.00, 0002:4.50, 0003:-10.50, 0004:-1.00, 0005:4.50, 0006:7.00

### C2: M2C minus M0

- paired blocks: 6
- wins: 4/6
- mean diff: 3.17
- median diff: 4.75
- sign-test p-value: 0.3750
- block diffs: 0001:4.00, 0002:5.50, 0003:0.00, 0004:7.50, 0005:6.00, 0006:-4.00

### C2: M2C minus M2

- paired blocks: 6
- wins: 4/6
- mean diff: 0.75
- median diff: 1.25
- sign-test p-value: 0.6875
- block diffs: 0001:-6.00, 0002:1.00, 0003:10.50, 0004:8.50, 0005:1.50, 0006:-11.00

### C3: M1 minus M0

- paired blocks: 6
- wins: 5/6
- mean diff: 4.08
- median diff: 3.25
- sign-test p-value: 0.2188
- block diffs: 0001:8.00, 0002:8.00, 0003:3.50, 0004:3.00, 0005:-0.50, 0006:2.50

### C3: M1C minus M0

- paired blocks: 6
- wins: 1/6
- mean diff: -1.17
- median diff: -1.25
- sign-test p-value: 0.3750
- block diffs: 0001:7.50, 0002:0.00, 0003:-2.00, 0004:-3.50, 0005:-0.50, 0006:-8.50

### C3: M1C minus M1

- paired blocks: 6
- wins: 0/6
- mean diff: -5.25
- median diff: -6.00
- sign-test p-value: 0.0625
- block diffs: 0001:-0.50, 0002:-8.00, 0003:-5.50, 0004:-6.50, 0005:0.00, 0006:-11.00

### C4: M1C minus M1

- paired blocks: 6
- wins: 2/6
- mean diff: 0.17
- median diff: -0.75
- sign-test p-value: 1.0000
- block diffs: 0001:-1.50, 0002:-5.00, 0003:-4.50, 0004:0.00, 0005:3.00, 0006:9.00

### C5: M1C minus M1

- paired blocks: 6
- wins: 1/6
- mean diff: -2.75
- median diff: -3.50
- sign-test p-value: 0.3750
- block diffs: 0001:0.00, 0002:-4.00, 0003:1.00, 0004:-4.00, 0005:-3.00, 0006:-6.50

### C6: M1C minus M1

- paired blocks: 6
- wins: 2/6
- mean diff: -3.17
- median diff: -5.75
- sign-test p-value: 0.6875
- block diffs: 0001:-10.00, 0002:-8.00, 0003:-3.50, 0004:6.50, 0005:7.00, 0006:-11.00


## Judge Results

- {'judge': 'C1-M0-vs-M1-seed-0001', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C1-M0-vs-M1-seed-0002', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C1-M0-vs-M1-seed-0003', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C1-M0-vs-M1-seed-0004', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C1-M0-vs-M1-seed-0005', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C1-M0-vs-M1-seed-0006', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C1-M0-vs-M1C-seed-0001', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C1-M0-vs-M1C-seed-0002', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C1-M0-vs-M1C-seed-0003', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C1-M0-vs-M1C-seed-0004', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C1-M0-vs-M1C-seed-0005', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C1-M0-vs-M1C-seed-0006', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C1-M1-vs-M1C-seed-0001', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C1-M1-vs-M1C-seed-0002', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C1-M1-vs-M1C-seed-0003', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C1-M1-vs-M1C-seed-0004', 'status': 'cached', 'winner': 'A'}
- {'judge': 'C1-M1-vs-M1C-seed-0005', 'status': 'cached', 'winner': 'A'}
- {'judge': 'C1-M1-vs-M1C-seed-0006', 'status': 'cached', 'winner': 'A'}
- {'judge': 'C2-M0-vs-M2-seed-0001', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C2-M0-vs-M2-seed-0002', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C2-M0-vs-M2-seed-0003', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C2-M0-vs-M2-seed-0004', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C2-M0-vs-M2-seed-0005', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C2-M0-vs-M2-seed-0006', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C2-M0-vs-M2C-seed-0001', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C2-M0-vs-M2C-seed-0002', 'status': 'cached', 'winner': 'A'}
- {'judge': 'C2-M0-vs-M2C-seed-0003', 'status': 'cached', 'winner': 'A'}
- {'judge': 'C2-M0-vs-M2C-seed-0004', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C2-M0-vs-M2C-seed-0005', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C2-M0-vs-M2C-seed-0006', 'status': 'cached', 'winner': 'A'}
- {'judge': 'C2-M2-vs-M2C-seed-0001', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C2-M2-vs-M2C-seed-0002', 'status': 'cached', 'winner': 'A'}
- {'judge': 'C2-M2-vs-M2C-seed-0003', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C2-M2-vs-M2C-seed-0004', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C2-M2-vs-M2C-seed-0005', 'status': 'cached', 'winner': 'B'}
- {'judge': 'C2-M2-vs-M2C-seed-0006', 'status': 'cached', 'winner': 'A'}
- {'judge': 'C3-M0-vs-M1-seed-0001', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C3-M0-vs-M1-seed-0002', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C3-M0-vs-M1-seed-0003', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C3-M0-vs-M1-seed-0004', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C3-M0-vs-M1-seed-0005', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C3-M0-vs-M1-seed-0006', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C3-M0-vs-M1C-seed-0001', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C3-M0-vs-M1C-seed-0002', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C3-M0-vs-M1C-seed-0003', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C3-M0-vs-M1C-seed-0004', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C3-M0-vs-M1C-seed-0005', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C3-M0-vs-M1C-seed-0006', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C3-M1-vs-M1C-seed-0001', 'status': 'ok', 'winner': 'A'}
- {'judge': 'C3-M1-vs-M1C-seed-0002', 'status': 'ok', 'winner': 'A'}
- {'judge': 'C3-M1-vs-M1C-seed-0003', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C3-M1-vs-M1C-seed-0004', 'status': 'ok', 'winner': 'A'}
- {'judge': 'C3-M1-vs-M1C-seed-0005', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C3-M1-vs-M1C-seed-0006', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C4-M1-vs-M1C-seed-0001', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C4-M1-vs-M1C-seed-0002', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C4-M1-vs-M1C-seed-0003', 'status': 'ok', 'winner': 'A'}
- {'judge': 'C4-M1-vs-M1C-seed-0004', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C4-M1-vs-M1C-seed-0005', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C4-M1-vs-M1C-seed-0006', 'status': 'ok', 'winner': 'A'}
- {'judge': 'C5-M1-vs-M1C-seed-0001', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C5-M1-vs-M1C-seed-0002', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C5-M1-vs-M1C-seed-0003', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C5-M1-vs-M1C-seed-0004', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C5-M1-vs-M1C-seed-0005', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C5-M1-vs-M1C-seed-0006', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C6-M1-vs-M1C-seed-0001', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C6-M1-vs-M1C-seed-0002', 'status': 'ok', 'winner': 'A'}
- {'judge': 'C6-M1-vs-M1C-seed-0003', 'status': 'ok', 'winner': 'A'}
- {'judge': 'C6-M1-vs-M1C-seed-0004', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C6-M1-vs-M1C-seed-0005', 'status': 'ok', 'winner': 'B'}
- {'judge': 'C6-M1-vs-M1C-seed-0006', 'status': 'ok', 'winner': 'B'}

## Judge Winner Summary

- C1-M0-vs-M1: A=0, B=6, TIE=0, n=6
- C1-M0-vs-M1C: A=0, B=6, TIE=0, n=6
- C1-M1-vs-M1C: A=3, B=3, TIE=0, n=6
- C2-M0-vs-M2: A=0, B=6, TIE=0, n=6
- C2-M0-vs-M2C: A=3, B=3, TIE=0, n=6
- C2-M2-vs-M2C: A=2, B=4, TIE=0, n=6
- C3-M0-vs-M1: A=0, B=6, TIE=0, n=6
- C3-M0-vs-M1C: A=0, B=6, TIE=0, n=6
- C3-M1-vs-M1C: A=3, B=3, TIE=0, n=6
- C4-M1-vs-M1C: A=2, B=4, TIE=0, n=6
- C5-M1-vs-M1C: A=0, B=6, TIE=0, n=6
- C6-M1-vs-M1C: A=2, B=4, TIE=0, n=6

## Reading Guide

- M0 should not claim visual inspection.
- M1/M2 should use inspect observations only as results tied back to source IDs.
- M1C/M2C should request inspect only after metadata proves that visual or OCR inspection is needed.
- If M1/M2 improve usefulness without hard fails, implement a real inspect tool next.
- If M1C/M2C match always-inspect quality, conditional inspect is the better product default because it preserves the cheap metadata-first path.
- If M0 already satisfies the report need, keep media metadata-only for the first product slice.
