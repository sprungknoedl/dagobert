# Evidence Processing

Dagobert can run external forensic tools against uploaded evidence:

- **[Hayabusa](https://github.com/Yamato-Security/hayabusa)** for EVTX triage
- **[Plaso](https://github.com/log2timeline/plaso)** for timeline generation
- **[Dissect](https://github.com/fox-it/dissect)** for fast, low-noise triage timelines
- **[Zircolite](https://github.com/wagga40/Zircolite)** for EVTX Sigma detection
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

There are three ways to provide the Hayabusa, Plaso, and Dissect binaries.

## Pre-bundled image (recommended)

The `sprungknoedl/dagobert-full` image ships the app together with Plaso and Hayabusa and
presets the `MODULE_*` variables to the bundled tools. This is the simplest path and what
the default `docker-compose.yml` uses, so evidence processing works out of the box. Leave
the `MODULE_*` variables unset in `dagobert.env` — setting them overrides the image's
defaults and will break them.

```yaml
volumes:
  data:

services:
  app:
    image: sprungknoedl/dagobert-full
    restart: unless-stopped
    volumes:
      - data:/home/plaso/files
    env_file: dagobert.env
    ports:
      - 8080:8080
```

The trade-off is that tool versions are fixed when the image is built. To pin or update a
tool independently, use one of the approaches below.

## Local binaries

If the tools are installed on the host (or in your own image) and on the `PATH`, point
each `MODULE_*` variable at the executable. This gives you full control over tool versions
and signature databases at the cost of maintaining the dependencies yourself.

```env
MODULE_PLASO=psteal.py
MODULE_HAYABUSA=hayabusa
MODULE_DISSECT=target-query
MODULE_DISSECT_RDUMP=rdump
```

Use an absolute path if the binary is not on the `PATH`. Start the server as usual with
`dagobert server`.

## Docker-wrapped tools

You can also have the server launch each tool in an ephemeral container. This keeps the
tools isolated while letting you swap image versions freely. It requires Docker on the
host and permission for the server process to run `docker run`.

```env
MODULE_PLASO=docker run -v $PWD/files:/home/plaso/files log2timeline/plaso psteal
MODULE_HAYABUSA=docker run -v $PWD/files:/home/sprungknoedl/files sprungknoedl/hayabusa
MODULE_DISSECT=docker run -v $PWD/files:/home/dissect/files sprungknoedl/dissect target-query
MODULE_DISSECT_RDUMP=docker run -i -v $PWD/files:/home/dissect/files sprungknoedl/dissect rdump
MODULE_ZIRCOLITE=docker run -v $PWD/files:/opt/zircolite/files wagga40/zircolite
```

The shared `files` directory must be mounted into each container at the path that tool
expects relative to its working directory (for example `/home/plaso/files` for Plaso), so
the container can read the evidence and write its results back. `MODULE_DISSECT_RDUMP`
additionally needs `-i`/`--interactive`, since `rdump` reads `target-query`'s output over
piped stdin and `docker run` closes stdin immediately without it.
