# Benchmarks

The Phase 3 benchmark suite covers tag parsing, normalized contract resolution,
deep promotion, the complete semantic rule registry, diagnostic fingerprinting,
and evolution comparison at 100 and 10,000 contracts.

Run it with:

```sh
go test -run '^$' -bench . -benchmem ./tag ./contract ./rules ./baseline ./evolution
```

Reference results from 2026-08-02 on Windows/amd64, Go 1.26.3, and an AMD Ryzen
9 9950X are shown below. They are observations from one development machine,
not performance guarantees.

| Benchmark | Time | Bytes/op | Allocations/op |
| --- | ---: | ---: | ---: |
| Parse struct tag | 485.2 ns | 1,232 | 15 |
| Resolve medium struct | 19.785 µs | 40,256 | 582 |
| Resolve heavily embedded struct | 30.116 µs | 45,304 | 917 |
| Complete rule registry | 40.641 µs | 39,844 | 706 |
| Fingerprint diagnostic | 268.9 ns | 216 | 6 |
| Compare 100 contracts | 63.613 µs | 67,545 | 950 |
| Compare 10,000 contracts | 7.733 ms | 7,057,857 | 90,263 |
| Build 32-field contract snapshot | 36.917 µs | 59,631 | 310 |
| Canonicalize snapshot | 314.8 ns | 448 | 2 |
| Fingerprint 32-field contract | 6.802 µs | 4,976 | 3 |
| Generate schema | 955.5 ns | 3,032 | 28 |
| Resolve JSON v1 and v2 profiles | 4.642 µs | 9,008 | 102 |
| Compare JSON profiles | 11.614 µs | 19,020 | 196 |
| Orchestrate two Git snapshots | 369.942 ms | 1,704,872 | 18,709 |

The numbers intentionally include allocations made by immutable result models.
Future optimization should use these benchmarks to prevent regressions while
preserving deterministic output and analyzer isolation.
