# 🚢 Releasing

## Cutting a release

Versions follow [semantic versioning](https://semver.org) and are git tags (`v0.2.0`). Pushing a
tag does most of the work:

```bash
git tag -a v0.2.0 -m "diskscan v0.2.0"
git push origin v0.2.0
```

Then attach the macOS archives (see below).

## The pipelines

| Workflow | Runs on | What it does |
|---|---|---|
| [`ci.yml`](../.github/workflows/ci.yml) | every push to `main`, every pull request | Vets and tests on Linux, builds the window and draws one frame of it under Xvfb. The frame is kept as the `window-linux` artifact. |
| [`release.yml`](../.github/workflows/release.yml) | every `v*` tag | Runs CI, builds both programs natively on Linux amd64 and arm64 runners, checks each window draws, and publishes a GitHub release with checksums. |

The release text comes from [`.github/release-notes.md`](../.github/release-notes.md), followed by
GitHub's generated list of changes. If the release already exists (for example, you created it by
hand), the workflow replaces its files instead of failing.

## macOS archives

macOS is not in the pipeline yet, because GitHub's macOS runners can take a long time to become
available. Build the archives on a Mac, from the tagged commit:

```bash
git checkout v0.2.0
scripts/release.sh v0.2.0
gh release upload v0.2.0 dist/*macos*.zip
```

[`scripts/release.sh`](../scripts/release.sh) builds each program for Apple Silicon and Intel,
joins them into one universal binary with `lipo`, targets macOS 13, signs it ad hoc and zips it
into `dist/`. It also builds the Linux web version, which the pipeline already publishes.

> [!IMPORTANT]
> After uploading, refresh `checksums.txt` in the release so it covers the macOS zips too. It is
> generated in the pipeline from the Linux files only.

## Screenshots

[`scripts/screenshots.sh`](../scripts/screenshots.sh) regenerates every picture in `docs/images`
(light and dark, window and web page). It needs a Mac.

It builds a made-up folder of sparse files at `/Users/Shared/keeper`. They look like 56 GiB but take
no disk space, and no real folder ever shows up in the pictures. The window is captured with
`-capture` and the web page with a headless Chromium (Playwright's, or set `CHROME`).

```bash
scripts/screenshots.sh
```
