
## 2026-04-24 - [Integer Overflow in Exponential Backoff]
**Vulnerability:** Integer overflow in exponential backoff delay calculation (`1<<uint(attemptCount)`) leading to zero or incorrect negative-like shifts when attempt count exceeds max bit size.
**Learning:** Operations involving bit shifting with unconstrained integers from database inputs (like attempt counts) can overflow and cause unintended logic states like zero delay loops.
**Prevention:** Always bound the shift amount before the shift operation (`if shift > 30 { shift = 30 }`).
