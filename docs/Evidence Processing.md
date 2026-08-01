# Evidence Processing

Dagobert can run external forensic tools against uploaded evidence:

- **[Hayabusa](https://github.com/Yamato-Security/hayabusa)** for EVTX triage
- **[Plaso](https://github.com/log2timeline/plaso)** for timeline generation
- **[Dissect](https://github.com/fox-it/dissect)** for fast, low-noise triage timelines
- **[Zircolite](https://github.com/wagga40/Zircolite)** for EVTX Sigma detection
- **[Chainsaw](https://github.com/WithSecureLabs/chainsaw)** for EVTX Sigma and
  native-rule hunting
- A built-in **[Timesketch](https://github.com/google/timesketch)** importer

These run as jobs in an in-process worker pool — part of the main `dagobert server`
process. There is no separate worker service to deploy or register: the server polls its
own jobs table and shells out to the configured tool for each job.

## How modules are configured

Each external tool is invoked through a command held in an environment variable:

- `MODULE_HAYABUSA` — command that runs Hayabusa
- `MODULE_PLASO` — command that runs Plaso's `psteal`
- `MODULE_DISSECT` — command that runs Dissect's `target-query`
- `MODULE_DISSECT_RDUMP` — command that runs Dissect's `rdump`, its companion tool for
  converting `target-query`'s output into Timesketch-ready JSONL
- `MODULE_ZIRCOLITE` — command that runs Zircolite
- `MODULE_CHAINSAW` — command that runs Chainsaw

Dagobert pipes `target-query`'s output into `rdump` (`target-query | rdump`) to build a
timeline; `MODULE_DISSECT` and `MODULE_DISSECT_RDUMP` configure the two commands
independently. `pip install dissect.target` installs both `target-query` and `rdump`
together, so for local (non-Docker) binaries the two variables just point at the same
install.

At startup the server validates each module (for example, by running the tool with a
`help`/version flag). Only modules that validate successfully accept jobs; the rest are
disabled, and the result for every module is shown on the settings page. If none are
configured, the server logs a warning and runs without evidence processing.

`DAGOBERT_WORKERS` (default `3`) sets how many jobs run concurrently.

The Timesketch importer is built into the app and needs no `MODULE_*` variable — it is
enabled by setting `TIMESKETCH_URL` (see [Configuration](Configuration.md)).

There are two ways to provide the tool binaries.

## Pre-bundled image (recommended)

The `sprungknoedl/dagobert` image ships the app together with all five tools — Plaso,
Hayabusa, Dissect, Zircolite, and Chainsaw — and presets the `MODULE_*` variables to the
bundled tools. This is the simplest path, so evidence processing works out of the box.
Leave the `MODULE_*` variables unset in `dagobert.env` — setting them overrides the
image's defaults and will break them.

```sh
docker run -d --name dagobert --restart unless-stopped \
  --env-file dagobert.env -v data:/home/dagobert/data -p 8080:8080 \
  sprungknoedl/dagobert
```

The trade-off is that tool versions are fixed when the image is built. To pin or update a
tool independently, provide it as a local binary instead.

## Local binaries

If the tools are installed on the host (or in your own image) and on the `PATH`, point
each `MODULE_*` variable at the executable. This gives you full control over tool versions
and signature databases at the cost of maintaining the dependencies yourself.

```env
MODULE_PLASO=psteal.py
MODULE_HAYABUSA=hayabusa
MODULE_DISSECT=target-query
MODULE_DISSECT_RDUMP=rdump
MODULE_ZIRCOLITE=zircolite
MODULE_CHAINSAW=chainsaw
```

Use an absolute path if the binary is not on the `PATH`. Start the server as usual with
`dagobert server`.

Chainsaw ships no Sigma rules, EVTX field mapping, or native rules of its own. `dagobert
update` fetches them into `external/chainsaw/` (the same convention as `external/mitre/` for
MITRE ATT&CK data), provided `MODULE_CHAINSAW` is set at the time it runs:

- `external/chainsaw/sigma_rules/` — the `rules/` tree from
  [SigmaHQ/sigma](https://github.com/SigmaHQ/sigma)'s `master` branch
- `external/chainsaw/mappings/sigma-event-logs-all.yml` — the mapping file from
  [Chainsaw's own repo](https://github.com/WithSecureLabs/chainsaw/tree/master/mappings)
- `external/chainsaw/rules/` — optional; [Chainsaw's bundled native
  rules](https://github.com/WithSecureLabs/chainsaw/tree/master/rules). If missing, Chainsaw
  just runs without its native rule set.

These always fetch the `master` branch and overwrite unconditionally — there is no version
pinning or staleness check, unlike MITRE's pinned release.

Zircolite likewise needs vendor assets it doesn't ship as a pip-installable package.
`dagobert update` fetches them into `external/zircolite/` from
[wagga40/Zircolite](https://github.com/wagga40/Zircolite)'s pinned `v3.7.6` tag, provided
`MODULE_ZIRCOLITE` is set at the time it runs:

- `external/zircolite/exportForTimesketch.tmpl` — the Timesketch export template
- `external/zircolite/rules_windows_generic.json` — Zircolite's default EVTX ruleset
- `external/zircolite/config.yaml` — the EVTX field mapping; mandatory, Zircolite produces
  no output without it
- `external/zircolite/transforms/` — referenced by `config.yaml`'s `transforms_dir:`, so it
  must stay a sibling of `config.yaml`

Unlike Chainsaw, this is pinned to a tag rather than tracking `master`: the template reads
Zircolite's own result structure, so a drifting template against a pinned tool would break
quietly.
