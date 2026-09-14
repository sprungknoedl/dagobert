# Contributing to Dagobert

## Issues

Bug reports, ideas, and questions are all welcome. Please search
[existing issues](https://github.com/sprungknoedl/dagobert/issues) first.

## Pull requests

When deciding whether to merge a pull request, here's what matters:

### Does it state intent

Explain the problem you're solving, not just what you changed. "Fix X" tells
me less than "Fix X because Y".

### Is it of good quality

`make check` passes — CI runs the same command, so nothing merges until it
does.

### Does it move the project closer to its vision

Dagobert is a single Go binary backed by SQLite, no separate services, KISS
and YAGNI over speculative flexibility. See the
[Development](README.md#development) section of the README for the build
setup and code layout.

### Does it follow the code of conduct

This project has a [Code of Conduct](CODE_OF_CONDUCT.md); contributions that
don't respect it will be rejected.
