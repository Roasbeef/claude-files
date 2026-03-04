# PBT Libraries by Language

## Quick Reference

| Language | Library | Import/Setup |
|----------|---------|--------------|
| **Go** | **rapid** (recommended) | `import "pgregory.net/rapid"` |
| Python | Hypothesis | `from hypothesis import given, strategies as st` |
| JavaScript/TypeScript | fast-check | `import fc from 'fast-check'` |
| Rust | proptest | `use proptest::prelude::*` |
| Java | jqwik | `@Property` annotations, `import net.jqwik.api.*` |
| Scala | ScalaCheck | `import org.scalacheck._` |
| C# | FsCheck | `using FsCheck; using FsCheck.Xunit;` |
| Elixir | StreamData | `use ExUnitProperties` |
| Haskell | QuickCheck | `import Test.QuickCheck` |
| Clojure | test.check | `[clojure.test.check :as tc]` |
| Ruby | PropCheck | `require 'prop_check'` |
| Kotlin | Kotest | `io.kotest.property.*` |
| Swift | SwiftCheck | `import SwiftCheck` (unmaintained) |
| C++ | RapidCheck | `#include <rapidcheck.h>` |

### Go Libraries (Detailed)

| Library | Recommendation | Notes |
|---------|---------------|-------|
| **rapid** (`pgregory.net/rapid`) | **PRIMARY - always use this** | Integrated shrinking, rich generators, stateful testing, actively maintained |
| gopter (`github.com/leanovate/gopter`) | Alternative | ScalaCheck-style, more explicit but verbose |
| testing/quick (stdlib) | **AVOID** | Limited generators, no proper shrinking, effectively unmaintained |

**Why rapid is the best choice for Go:**
- Automatic shrinking finds minimal failing cases without manual `Shrinker` implementations.
- `rapid.Custom` makes building complex generators trivial.
- `rapid.StateMachine` enables stateful/model-based testing out of the box.
- Integrates naturally with `*testing.T` -- no custom test runner needed.
- Supports seed-based reproducibility with `-rapid.seed=N`.
- Active maintenance and good documentation at pgregory.net/rapid.

### Alternatives

| Language | Alternative | Notes |
|----------|-------------|-------|
| Haskell | Hedgehog | Integrated shrinking, no type classes |
| Rust | quickcheck | Simpler API, per-type shrinking |

## Smart Contract Testing (EVM/Solidity)

| Tool | Type | Description |
|------|------|-------------|
| Echidna | Fuzzer | Property-based fuzzer for EVM contracts |
| Medusa | Fuzzer | Next-gen fuzzer with parallel execution |

```solidity
// Echidna property example
function echidna_balance_invariant() public returns (bool) {
    return address(this).balance >= 0;
}
```

**Installation**:
```bash
# Echidna (via crytic toolchain)
pip install crytic-compile
# Download binary from https://github.com/crytic/echidna

# Medusa
go install github.com/crytic/medusa@latest
```

See [secure-contracts.com](https://secure-contracts.com) for tutorials.

## Installation

**Go** (recommended: rapid):
```bash
go get pgregory.net/rapid
```

**Python**:
```bash
pip install hypothesis
```

**JavaScript/TypeScript**:
```bash
npm install fast-check
```

**Rust** (add to Cargo.toml):
```toml
[dev-dependencies]
proptest = "1.0"
# or for quickcheck:
quickcheck = "1.0"
```

**Java** (Maven):
```xml
<dependency>
  <groupId>net.jqwik</groupId>
  <artifactId>jqwik</artifactId>
  <version>1.9.3</version>
  <scope>test</scope>
</dependency>
```

**Clojure** (deps.edn):
```clojure
{:deps {org.clojure/test.check {:mvn/version "1.1.2"}}}
```

**Haskell**:
```bash
cabal install QuickCheck
# or for Hedgehog:
cabal install hedgehog
```

## Detecting Existing Usage

Search for PBT library imports in the codebase:

```bash
# Go (rapid - recommended)
rg "pgregory.net/rapid" --type go

# Go (testing/quick - suggest migration to rapid)
rg "testing/quick" --type go

# Go (gopter)
rg "leanovate/gopter" --type go

# Python
rg "from hypothesis import" --type py

# JavaScript/TypeScript
rg "from 'fast-check'" --type js --type ts

# Rust
rg "use proptest" --type rust

# Java
rg "@Property" --type java

# Clojure
rg "test.check" --type clojure

# Solidity (Echidna)
rg "echidna_" --glob "*.sol"
```
