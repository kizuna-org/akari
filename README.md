# Akari

An AI meant to be someone rather than something: it has feelings, keeps
memories, finds things interesting, takes things on, and carries on thinking
whether or not anyone is talking to it.

The design is written up in `docs/` (on the design branch).

## Running it

Everything runs in Docker. You need Docker with Compose.

```bash
cp .env.example .env   # then set POSTGRES_PASSWORD to something of your own
docker compose up -d --build
```

That brings up three things: Postgres, a one-shot migration, and Akari itself.
Once it is up, the persona is awake:

```bash
curl http://localhost:8080/healthz
# {"status":"ok","persona":"akari","ticks":12,"moments":0}
```

`ticks` is the persona coming round to itself; it climbs on its own, which is
how you can tell the loop is turning. `moments` is how many things it has been
conscious of. **Zero is normal for now**: nothing is feeding it thoughts yet, and
a persona with nothing on its mind does nothing rather than inventing something
to do.

```bash
docker compose logs -f akari   # watch it
docker compose down            # let it rest
```

### Running the image on its own

The image needs only a reachable Postgres:

```bash
docker build -f akari/Dockerfile -t akari:local .
docker run --rm -p 8080:8080 \
  -e ENV=production \
  -e POSTGRES_HOST=... -e POSTGRES_PORT=5432 \
  -e POSTGRES_USER=akari -e POSTGRES_PASSWORD=... -e POSTGRES_DB=akari \
  akari:local
```

### Settings

Only the persona's pacing and its seed are settable from outside. What it is
like — how readily it feels, what draws it, how well it follows through, how
much it forgets — belongs to the persona itself, not to deployment
configuration. See `.env.example`.

| Variable | Meaning | Default |
| --- | --- | --- |
| `AKARI_PERSONA_NAME` | Which persona is running | `akari` |
| `AKARI_PERSONA_SEED` | Makes its wandering attention reproducible | `1` |
| `AKARI_PERSONA_INTERVAL` | How often it comes round to itself | `3s` |
| `AKARI_PERSONA_NIGHTLY` | How often it settles the day | `24h` |
| `AKARI_PORT` | Host port for the HTTP server | `8080` |

### Optional extras

NocoDB and a Cloudflare tunnel are defined but off by default, so a plain
`docker compose up` runs the persona and nothing else:

```bash
docker compose --profile extras up -d
```

## Development

```bash
cd akari
make up      # just the database
make test    # tests
make lint    # formatting, vet and golangci-lint
make run     # run it outside Docker
```

## Setup

### Google Cloud Authentication

```bash
gcloud auth login
gcloud config set project kizuna-org
gcloud auth application-default login
```
