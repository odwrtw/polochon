// Package simkl implements Polochon's Explorer, Searcher, Detailer and Calendar
// interfaces using the public Simkl movie and TV APIs. The separate anime
// catalog, user lists, sync, and scrobbling are deliberately unsupported.
//
// Configuration requires client_id from https://simkl.com/settings/developer/.
// No client secret, user credentials, OAuth flow or bearer token is used.
// app_version is optional (default "dev"); set it to the application's version
// when distributing a release. All requests identify the app as "polochon".
//
// Exploration returns up to 40 entries, search up to 20, preserving source order.
// Only the first page is fetched because Polochon's interfaces have no paginator.
// Search and sparse discovery entries are enriched with cached detail lookups
// to obtain the IMDb IDs required by Polochon. Unidentified items are omitted.
// Runtime display strings in trending feeds are not parsed; details supply minutes.
//
// Movie options: trending (today), trending-week, trending-month, dvd,
// popular-this-week, popular-this-month, popular-all-time, rank, release-date,
// voted, watched. TV supports the same options except dvd, plus best, best-year,
// best-month, best-voted, best-watched, premieres, upcoming, airing (today), and
// airing-tomorrow. There are no aliases for unsupported Trakt-only lists.
//
// Details preserve fields Simkl cannot provide, including external episode IDs.
// Unnumbered specials are omitted rather than assigned invented coordinates.
// Show banners use Simkl's landscape poster crop. Images use the recommended
// wsrv.nl proxy with JPEG output to match Polochon's existing sidecar files.
// Calendar lookups do not require artwork. Undated unaired episodes are omitted
// because Polochon otherwise considers an unknown air date already available.
//
// Consumers displaying Simkl data must comply with https://api.simkl.org/api-rules,
// including source attribution and naming Simkl in trending sections. Show.URL
// contains a per-title source link; Polochon's Movie type has no source-link field.
// Public UI attribution and any necessary permission for metadata use remain the
// consuming application's responsibility, not an authentication feature here.
//
// API contracts: https://api.simkl.org/all-endpoints,
// https://api.simkl.org/conventions/headers,
// https://api.simkl.org/conventions/null-values,
// https://api.simkl.org/resources/rate-limits and https://wsrv.nl/docs/format.
package simkl
