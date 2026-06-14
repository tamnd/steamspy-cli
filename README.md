# steamspy

SteamSpy — Steam game ownership and player stats

`steamspy` is a single pure-Go binary. It reads public steamspy data
over plain HTTPS, shapes it into clean records, and prints output that pipes
into the rest of your tools. No API key, nothing to run alongside it.

The same package is also a [resource-URI driver](#use-it-as-a-resource-uri-driver),
so a host program like [ant](https://github.com/tamnd/ant) can address
steamspy as `steamspy://` URIs.

## Install

```bash
go install github.com/tamnd/steamspy-cli/cmd/steamspy@latest
```

Or grab a prebuilt binary from the [releases](https://github.com/tamnd/steamspy-cli/releases), or run
the container image:

```bash
docker run --rm ghcr.io/tamnd/steamspy:latest --help
```

## Usage

```bash
steamspy page <path>                      # fetch one page as a record
steamspy page <path> -o json              # as JSON, ready for jq
steamspy page <path> --template '{{.Body}}'  # just the readable body text
steamspy links <path>                     # the pages it links to, one per line
steamspy --help                           # the whole command tree
```

Every command shares one output contract: `-o table|json|jsonl|csv|tsv|url|raw`,
`--fields` to pick columns, `--template` for a custom line, and `-n` to limit.
The default adapts to where output goes (a table on a terminal, JSONL in a
pipe), so the same command reads well by hand and parses cleanly downstream.

This is a fresh scaffold. It ships one example resource type, `page`, wired end
to end. Model the real steamspy records in `steamspy/` and declare their
operations in `steamspy/domain.go`; each one becomes a command, an HTTP
route, and an MCP tool at once.

## Serve it

The same operations are available over HTTP and as an MCP tool set for agents,
with no extra code:

```bash
steamspy serve --addr :7777    # GET /v1/page/<path>  returns NDJSON
steamspy mcp                   # speak MCP over stdio
```

## Use it as a resource-URI driver

`steamspy` registers a `steamspy` domain the way a program registers a
database driver with `database/sql`. A host enables it with one blank import:

```go
import _ "github.com/tamnd/steamspy-cli/steamspy"
```

Then [ant](https://github.com/tamnd/ant) (or any program that links the package)
dereferences `steamspy://` URIs without knowing anything about steamspy:

```bash
ant get steamspy://page/<path>   # fetch the record
ant cat steamspy://page/<path>   # just the body text
ant ls  steamspy://page/<path>   # the pages it links to, each addressable
ant url steamspy://page/<path>   # the live https URL
```

## Development

```
cmd/steamspy/   thin main: hands cli.NewApp to kit.Run
cli/                 assembles the kit App from the steamspy domain
steamspy/                the library: HTTP client, data models, and domain.go (the driver)
docs/                tago documentation site
```

```bash
make build      # ./bin/steamspy
make test       # go test ./...
make vet        # go vet ./...
```

## Releasing

Push a version tag and GitHub Actions runs GoReleaser, which builds the
archives, Linux packages, the multi-arch GHCR image, checksums, SBOMs, and a
cosign signature:

```bash
git tag v0.1.0
git push --tags
```

The Homebrew and Scoop steps self-disable until their tokens exist, so the first
release works with no extra secrets.

## License

Apache-2.0. See [LICENSE](LICENSE).
