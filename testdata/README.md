# Synthetic Fixtures

Every history and credential-shaped value here was invented for testing. No
credential was issued by a provider and no actual conversation was copied.
Do not contact any provider to verify these values.

`demo/` contains four schema-shaped examples with one seeded GitHub-token
pattern each. These are not recordings from real client versions. The demo
should produce four distinct occurrences, not one deduplicated result.

```sh
./bin/unspill scan --path ./testdata/demo
./bin/unspill scan --agent generic --path ./testdata/generic.json
```

Both commands intentionally return exit code 1 at the default high threshold.
Use `--fail-on none` when demonstrating without a findings failure exit.
Generic exports are intentionally not auto-detected.