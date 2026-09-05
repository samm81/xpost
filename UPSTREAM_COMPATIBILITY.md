# upstream compatibility references

the provider-specific text behavior in this fork is adapted from the upstream
snapshots below. these references are documentation only: builds do not fetch
or update them automatically.

## x text counting

- project: [twitter-text](https://github.com/twitter/twitter-text)
- snapshot: `30e2430d90cff3b46393ea54caf511441983c260` on `master`
- source: [`config/v3.json`](https://github.com/twitter/twitter-text/blob/30e2430d90cff3b46393ea54caf511441983c260/config/v3.json)
- local code: [`internal/xpost/twitter/twittertext.go`](internal/xpost/twitter/twittertext.go)

the local implementation follows the snapshot's weighted length of 280,
default and official character ranges, emoji parsing, and transformed URL
length of 23. it is an adaptation of the rules, not a copy of the complete
upstream parser. changes to the upstream configuration or URL detection rules
require a review of the local implementation and its tests.

## bluesky post limits

- project: [atproto](https://github.com/bluesky-social/atproto)
- snapshot: `96c843845fcc42ac2644abd7de06076abf467ac1` on `main`
- source: [`app.bsky.feed.post`](https://github.com/bluesky-social/atproto/blob/96c843845fcc42ac2644abd7de06076abf467ac1/lexicons/app/bsky/feed/post.json)
- local code: [`internal/xpost/bluesky/bluesky.go`](internal/xpost/bluesky/bluesky.go)

the local validation uses the snapshot's 300 grapheme and 3000 byte limits.

## bluesky link display shortening

- project: [social-app](https://github.com/bluesky-social/social-app)
- snapshot: `ee32926ed5f07f562087c1df1c90d739457e48d2` on `main`
- sources: [`url-helpers.ts`](https://github.com/bluesky-social/social-app/blob/ee32926ed5f07f562087c1df1c90d739457e48d2/src/lib/strings/url-helpers.ts), [`rich-text-manip.ts`](https://github.com/bluesky-social/social-app/blob/ee32926ed5f07f562087c1df1c90d739457e48d2/src/lib/strings/rich-text-manip.ts), and [`composer.ts`](https://github.com/bluesky-social/social-app/blob/ee32926ed5f07f562087c1df1c90d739457e48d2/src/view/com/composer/state/composer.ts)
- local code: [`internal/xpost/bluesky/blueskytext.go`](internal/xpost/bluesky/blueskytext.go)

the local implementation follows the web composer behavior of shortening the
visible URL before counting graphemes while keeping the original URL in the
link facet. this is client behavior, not a promise that the Bluesky protocol
will always use the same display form.

## transient operation retries

provider uploads and post creation retry up to three attempts for request
timeouts, rate limits, server errors, and transient network or incomplete
responses. retries use a bounded exponential backoff starting at 500 ms and
ending at 4 seconds. validation, authentication, and other permanent errors
are returned without retrying.

the provider APIs do not expose one shared idempotency mechanism. if a post is
accepted by a provider but its response is lost, a retry can create a
duplicate post.

## update policy

update these references deliberately when the provider behavior changes:

1. compare the new upstream source with the recorded snapshot;
2. update the local implementation and conformance tests together;
3. run the bridge checks and an authenticated provider test when available;
4. commit the fork change and then update the root repository's submodule
   pointer.

do not update these snapshots automatically. platform server behavior cannot be
pinned, so a provider-side change may still require a code update even when the
local submodule pointer is unchanged.
