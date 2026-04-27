## 2026-04-27 - Integer Overflow in Dynamic Backoff Calculations
**Vulnerability:** An integer overflow `int -> uint` vulnerability leading to an unconstrained bitwise shift (`1 << uint(attemptCount)`) was discovered in webhook delivery backoff retry delay calculations.
**Learning:** `attemptCount` increments can grow unboundedly during retry loops. Directly using user/database-controlled unconstrained numbers in bitwise shift operations (`<<`) will lead to integer overflow/underflow, runtime panics, or zero evaluations if the value exceeds bit bounds (e.g. `31` or `63`).
**Prevention:** Always cap variables used in dynamic bitwise shift calculations (e.g., `if shift > 30 { shift = 30 }`) and sanitize for underflows (`if shift < 0 { shift = 0 }`) before applying them to a shift operator.
