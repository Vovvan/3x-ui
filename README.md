# strip-sub-bots patches

Patch archive for the `3x-ui` fork that removes subscription server/bot
integrations, sponsor placements, multi-node management, and the scheduled
cache/mutation workflows. The upstream base is recorded in `PATCH-BASE`.

The series contains only code and documentation changes. Fork-only `.github`
workflow changes are excluded so the patches stay applicable to pristine
upstream.

## Apply to pristine upstream

```sh
git clone --branch v3.11.0 https://github.com/MHSanaei/3x-ui.git 3x-ui
./scripts/apply.sh 3x-ui
```

Or apply manually:

```sh
git -C 3x-ui am -3 "$PWD"/patches/*.patch
```

## Regenerate from the fork branch

```sh
./scripts/export.sh /path/to/3x-ui-fork v3.11.0
```

The export command excludes `.github` changes and rewrites `PATCH-BASE` and
`series`. Update both when the upstream base changes.

## Verify

Dispatch the `Patch verify (strip-sub-bots)` workflow from the default branch
with `base=v3.11.0` and `patch_ref=patches/strip-sub-bots`. The workflow applies
the series to pristine upstream, regenerates the OpenAPI schemas, builds the
frontend, and builds/tests the Go packages.

## Rebase flow

1. Rebase `fork/strip-sub-bots` onto the new upstream tag.
2. Run `scripts/export.sh` against the rebased branch.
3. Push the updated patch archive.
4. Dispatch `patch-verify` with the new base and `patches/strip-sub-bots`.
