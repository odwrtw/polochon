# Simkl movie and TV module

Simkl provides movie and TV discovery, details, aired-episode calendars, authenticated search, and a read-only Plan-to-Watch snapshot through Polochon's existing module interfaces. Anime and playback/account tracking writes are excluded.

## Public mode

Select the explorer without registering an app or adding any module parameters:

```yaml
movie:
  explorers: [simkl]
show:
  explorers: [simkl]
```

Merge these settings into your normal configuration, including existing library directories. No `modules_params` container, token file, cache directory, or writable credential storage is required for public operation.

Movie and TV options:

- `simkl-trending-today`, `simkl-trending-week`, `simkl-trending-month`: up to 100 entries in upstream watcher order.
- The same names with `-500`: up to 500 entries.
- `simkl-genre:<slug>`: up to 500 entries from that genre's monthly list. `AvailableMovieOptions()` and `AvailableShowOptions()` obtain supported slugs from the respective upstream indexes. If the index cannot be loaded, they return the built-in options.
- Movies additionally support `simkl-dvd` and `simkl-dvd-500`.

Items without usable IMDb identity and conflicting canonical records are skipped with an aggregate warning. Authenticated text search returns the first 20 upstream matches, hydrating missing IMDb IDs through cached public detail requests because SIMKL's search ID block omits them. Results use existing movie/show types; they have no pagination or source/freshness envelope. This module adds no HTTP discovery endpoints or Go API-client methods.

Known IDs from a discovery list can be detailed anonymously while their module mappings remain cached. Resolving arbitrary IMDb/TMDB/TVDB IDs after startup or cache eviction requires authentication. Missing connection returns `ErrConnectionRequired`; text search never silently falls back to filtering a small cached list.

Anonymous catalog access was observed on 28 October 2026. SIMKL's common request documentation still calls for a client ID. The module sends a real ID if configured and returns an upstream error if anonymous access is restricted later; it never invents or shares a client ID.

## Connect an account

Register your own **AUTH V2** app at [Simkl developer settings](https://simkl.com/settings/developer/). A public device/CLI registration requires no client secret or callback URL. Run:

```sh
go run ./modules/simkl/cmd/setup -client-id YOUR_SIMKL_V2_CLIENT_ID
```

Open the printed URL, approve `media:read`, and wait for verification. The helper refreshes the same credential through two fresh client instances to check that it survives cold starts without rewriting configuration. It stops if upstream rotates the refresh token or fails scope validation. It never saves a token file. Instructions go to stderr; the verified credential-bearing YAML is deliberately printed once to stdout.

For a confidential server registration, set `SIMKL_CLIENT_SECRET` in the helper's environment. The helper includes that secret in its output configuration. Keep the output and resulting configuration private. Do not include real credentials in examples or source control.

Merge the generated module entry into your existing `modules_params` list:

```yaml
modules_params:
  - name: simkl
    client_id: "YOUR_SIMKL_V2_CLIENT_ID"
    refresh_token: "YOUR_SIMKL_REFRESH_TOKEN"
    # client_secret: "YOUR_CONFIDENTIAL_CLIENT_SECRET"

movie:
  explorers: [simkl]
  detailers: [simkl]
  searchers: [simkl]
show:
  explorers: [simkl]
  detailers: [simkl]
  searchers: [simkl]
  calendar: simkl
wishlist:
  wishlisters: [simkl]
  movie_default_qualities: [1080p, 720p]
  show_default_qualities: [720p, 1080p]
```

Keep your existing torrenters, download client, directories, guessers, and subtitle configuration. Other module defaults are unchanged. One connected Simkl account is shared across these roles; Polochon HTTP API tokens do not select separate upstream accounts.

Use a separate authorization grant for each concurrently running Polochon instance: refreshing replaces the grant's previous access token immediately. Access tokens stay in memory and renew as needed, with one refresh shared across roles. The documented lifetime is seven days; refresh tokens are non-rotating with a sliding 180-day expiry. Revocation or six months without a successful refresh can require setup again. Actual expiry metadata controls renewal. No running test account is bundled; use the setup helper's verification before relying on authenticated operation.

Removing credentials and restarting returns to public mode and discards private state. Missing/revoked credentials, insufficient scope, or exhausted account quota affect authenticated operations while public discovery continues.

## Wishlists, calendars, and import limits

Plan-to-Watch movies and full TV shows are loaded once on the first wishlist access after process start. Movie and show reads share one complete account snapshot. Existing quality defaults and IMDb-based merging apply; TV tracking begins at season 1, episode 1. Entries with unsupported or conflicting identities are omitted with diagnostics. Source-scoped episode selections and upper episode bounds are unsupported.

**Restart Polochon to pick up remote watchlist additions, removals, or list moves.** Scheduled downloader reads reuse the process snapshot without remote polling. Snapshot time and counts are logged after success. Failed initialization backs off for a minute and publishes no partial snapshot. No scan/playback event hooks or server sync actions are added. Reaching an upstream item limit returns an explicit error rather than publishing a truncated wishlist.

Show metadata retains the representable episode catalog. The calendar is filtered inside this provider to include only positively dated, already-aired episodes. Future, explicitly unaired, unknown-date, or conflicting entries cannot reach the existing downloader through this calendar. Date-only releases become available after the full release day in the show's known timezone, including DST transitions. Historical aired episodes remain eligible for catch-up.

Native posters and episode images use SIMKL's documented WebP URLs; fanart uses its JPEG `_d` variant. Full upstream image URLs are preserved. Missing optional images never invalidate the provider's metadata response. There is no mandatory Fanart.tv dependency or artwork substitution.

**The existing library's image requirements still apply.** It requires movie poster/fanart and show poster/fanart/banner and can reject imports if images are missing or fail. SIMKL has no documented native banner family. Existing `.jpg` sidecars are not converted from WebP by this module. Use compatible existing artwork/enrichment where available; selecting this detailer does not guarantee that every title can be imported. No library error policy, NFO format, shared identity, downloader backend, or index behavior changes here.

## Cache, errors, and attribution

The module shares a sequential scheduler across roles (GET starts spaced at least 150 ms apart, POST starts at least one second apart), context-aware HTTP requests, a 20-second HTTP timeout, 16 MiB response limit, 128-response/32 MiB cache, and bounded internal ID mappings. Caches are memory-only. Discovery defaults to hourly freshness for today and daily freshness for other lists; details/episodes default to one hour. HTTP cache directives and validators take precedence, with a maximum cache lifetime of one day. Eligible public discovery data may be reused for up to one additional day on transient outages. Episode catalogs never use that stale fallback, and airing eligibility is checked on each calendar read.

Provider errors distinguish connection, scope/plan, quota, ambiguous/unsupported identity, not-found, and upstream failures. `APIError` carries HTTP status, a sanitized upstream code, and a retry time when known. Daily account limits reset at the next New York midnight. Missing quota headers are not treated as zero. Safe reads have bounded retries; OAuth exchanges are not blindly replayed after uncertain outcomes. Credentials never accompany public requests or redirects.

Consumers displaying this data must follow [SIMKL's attribution and API rules](https://api.simkl.org/api-rules): identify the source as Simkl, brand trending sections accordingly, and link items to their Simkl pages. `Show.URL` contains the canonical TV link in Go; existing Show JSON intentionally omits it. Current movie types have no source-URL field. A presentation layer can use [SIMKL's redirect deep links](https://api.simkl.org/api-reference/redirect) with existing external IDs for item links, without fetching those HTML destinations in the adapter. The existing types do not enforce attribution in a UI; integrating applications must provide it. SIMKL's commercial licensing and metadata-use terms also apply.

## Validation

```sh
go test ./modules/simkl/... ./lib/configuration
go test -race ./modules/simkl/... ./lib/configuration
```

Normal tests use local HTTP fixtures and injected time, including recorded public schemas, concurrency, refresh/cold-start simulation, snapshot reuse/failures, quota resets, calendar DST boundaries, and compatible imports/NFO index rebuilds through the unchanged library.

Optional anonymous production smoke test:

```sh
SIMKL_LIVE=1 go test ./modules/simkl -run TestLivePublicMovieTV -v -count=1
```

Live account authorization is performed explicitly by the setup helper. Neither ordinary tests nor the public smoke test require credentials or make tracking writes.
