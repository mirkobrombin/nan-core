<div align="center">
  <h1>nan-core</h1>
  <p>A deterministic reasoning engine that never invents a fact.</p>
  <p>
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go" alt="Go 1.25+">
    <img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT">
  </p>
</div>

You give it facts and rules. It answers a query with a truth value and the proof
that produced it. What it was never told and cannot derive is `unknown`, and it
says so instead of guessing. Two facts that disagree become a contradiction it
reports rather than a coin flip.

It is a library, not a service, and it holds no model. There is no LLM here and
no natural language: this is the symbolic core, the part that decides what is
true. The language layer that talks to it lives elsewhere.

## Install

```sh
go get github.com/mirkobrombin/nan-core@latest
```

## Packages

| package  | what it does                                                    |
| -------- | -------------------------------------------------------------- |
| `engine` | ingest, evaluate, derive; the entry point that ties the rest together |
| `belief` | atoms, polarity, and the store that detects a contradiction    |
| `rules`  | inference rules over atom patterns with `?`-prefixed variables |
| `kg`     | the knowledge graph the facts live in                          |
| `store`  | the append-only log a run replays from: a WAL, or slipstream   |
| `vsa`    | vector-symbolic similarity over atoms                          |
| `ilp`    | rule suggestions induced from recurring chains                 |

## Example

Record two facts and a transitive rule, then ask a question neither fact states
on its own:

```go
e, _ := engine.New()

e.Ingest(belief.Belief{
    Atom:     belief.Atom{From: "socrate", Predicate: "isa", To: "uomo"},
    Polarity: belief.PolarityPositive,
    Source:   "example",
})
e.Ingest(belief.Belief{
    Atom:     belief.Atom{From: "uomo", Predicate: "isa", To: "mortale"},
    Polarity: belief.PolarityPositive,
    Source:   "example",
})

e.AddRule(rules.Rule{
    Name: "isa-transitive",
    If: []rules.AtomPattern{
        {From: "?x", Predicate: "isa", To: "?y"},
        {From: "?y", Predicate: "isa", To: "?z"},
    },
    Then: rules.AtomPattern{From: "?x", Predicate: "isa", To: "?z"},
})

v, proof := e.Evaluate(belief.Atom{From: "socrate", Predicate: "isa", To: "mortale"})
// v is TruthTrue, proof.Kind is "derived"
```

A rule variable is a token that starts with `?`. A bare token is a literal node,
so `?x` matches anything and `socrate` matches only itself.

The full program is in [`examples/reason`](examples/reason/main.go); run it with
`go run ./examples/reason`.

## Truth

`Evaluate` returns one of four values, because "not true" and "unknown" are not
the same answer:

| value          | meaning                                        |
| -------------- | ---------------------------------------------- |
| `TruthUnknown` | no fact and no rule reaches it                 |
| `TruthTrue`    | asserted, or derived                           |
| `TruthFalse`   | asserted with negative polarity                |
| `TruthBoth`    | asserted both ways, a contradiction to resolve |

`TruthBoth` is the point of the engine: it does not pick a side. `Resolve`
records which polarity wins, and the decision is in the log like any other fact.

## The log

State is not held in memory alone. Every belief, resolution and rule is appended
to a log, and `ReplayFromLog` rebuilds the engine from it, so a process restart
loses nothing. Two backends satisfy the same `store.Log` interface:

- `store.OpenWAL` writes a single append-only file.
- `store.OpenSlipstreamLog` writes through slipstream.

`store.LogBackends` is a registry, so an application picks a backend by name.

## License

MIT
