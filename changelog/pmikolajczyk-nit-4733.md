### Fixed
- Canonicalize NaN values returned by soft-float library functions so WAVM and JIT produce identical results when floating-point operations yield NaN
