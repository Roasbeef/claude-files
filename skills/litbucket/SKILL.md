---
name: litbucket
description: Publish, search, and inspect internal Litbucket static artifacts. Use when asked to publish a site, dashboard, notebook, report, or other static bundle to an internal address, to find one that already exists, or to share a generated artifact with the team. Triggers include "publish this to litbucket", "put this on lightning.wiki", "find the <name> dashboard", "give me a link to this report".
---

# Litbucket

Litbucket is Lightning Labs' internal shelf for small static artifacts: sites,
dashboards, notebooks, generated reports. Upload a ZIP with an `index.html` at
its root and the site gets one stable `<slug>.lightning.wiki` address plus one
immutable `v-<uuid>.litbucket.staging.lightningcluster.com` address per
publish. Litbucket never runs server code or installs dependencies from a
bundle; it stores and serves bytes.

## Install

The repository is private, so Go needs to fetch it over SSH rather than the
public module proxy. This is one-time setup:

```bash
go env -w GOPRIVATE=github.com/lightninglabs/*
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

Then install a pinned release and confirm it:

```bash
go install github.com/lightninglabs/litbucket/cmd/litbucket@v1.0.0
litbucket --version
```

`go install` writes to `GOBIN`, or `$(go env GOPATH)/bin` when `GOBIN` is
empty. If the binary is not found, that directory is missing from `PATH`.

From a clone, `make install` does the same thing from the working tree.

## Authenticate

Set the API endpoint, then exchange the identity `gh` already holds for a
one-hour Litbucket session:

```bash
export LITBUCKET_ENDPOINT=https://litbucket-api.staging.lightningcluster.com
litbucket --json auth login --gh --no-browser
litbucket --json auth status
```

Pass `--no-browser` even with a pseudo-terminal. It guarantees that a missing
GitHub login returns a structured action instead of opening a browser or
blocking on stdin.

If the command returns `github_auth_required`, show the user its `command` and
`verification_url`, then wait. Do not run an interactive browser flow on their
behalf. Retry the same Litbucket login after they authenticate `gh`.

The login sends the `gh` credential only to the GitHub exchange endpoint and
stores only the resulting narrow Litbucket session, in Keychain on macOS or an
owner-only file elsewhere. CI without a GitHub user sets `LITBUCKET_TOKEN`
instead, which takes precedence. Never pass a token as a command-line argument
and never print one.

## Read the contract before publishing

```bash
litbucket --json schema
```

This returns the live field names, sort values, limits, and authorization
rules. Read it rather than assuming the shape of a publish request.

## Publish

Validate first. A dry run does no network write and no catalog change:

```bash
litbucket --json publish site.zip \
  --name "Explorer dashboard" \
  --team engineering \
  --dry-run
```

Publish only after validation succeeds:

```bash
litbucket --json publish site.zip \
  --name "Explorer dashboard" \
  --slug explorer-dashboard \
  --team engineering \
  --description "Internal network visibility."
```

Report the returned `vanity_url` as the team-facing link and `version_url` as
the link pinned to those exact bytes.

Publishing the same slug again appends a version and advances the vanity
address. The publisher must belong to the requested GitHub team, and updating
an existing site also requires membership in that site's current team.
Republishing identical bytes to one site is rejected as a no-op.

## Search and inspect

```bash
litbucket --json list --sort latest --limit 20
litbucket --json list --query explorer --sort latest
litbucket --json list --team engineering --sort name --limit 100
litbucket --json show <site-id>
```

Successful JSON goes to stdout, errors to stderr.

## Export an artifact to Markdown

The control UI at `https://litbucket.lightning.wiki` carries an export control
on every catalog row and in the site detail panel. It serves:

```text
GET /api/v1/sites/{id}/markdown[?version=<version-id>]
```

The document holds the site metadata, both addresses, the version ledger, and
a best-effort conversion of the page. A hand-written page, report, or set of
notes converts well. A dashboard drawn by JavaScript has no text in its entry
point, so its export carries the metadata and says so. There is no `litbucket
export` subcommand yet; point the user at the UI control.

## Failure contract

Exit codes are stable. Branch on them rather than parsing messages:

| Code | Meaning |
| ---: | --- |
| `0` | Success |
| `1` | Network, server, or unexpected failure |
| `2` | Invalid arguments or bundle |
| `3` | Authentication required |
| `4` | Site not found |
| `5` | Publish or vanity slug conflict |

Exit code 3 covers `github_auth_required`, `github_membership_required`, and
`github_scope_required`. Each carries a `command` the user should run.

## Rules

- Never extract an uploaded bundle locally. Let the server validate archive
  paths, size limits, symlinks, and the entry point.
- Never print, log, or echo a token or session.
- Always dry-run before a first publish to a new slug.
- Always request `--json`. Human-formatted output is not a stable contract.
