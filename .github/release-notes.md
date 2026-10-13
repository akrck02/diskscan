`diskscan` shows what is using your disk: scan a folder, then click through it as a sorted list or
a treemap. It comes as a desktop window (`diskscan-ui`) and as a local web page (`diskscan`).

**Linux (amd64, arm64)**

```bash
tar -xzf diskscan-ui-<version>-linux-amd64.tar.gz
./diskscan-ui ~
```

- `diskscan-ui` needs OpenGL 4.1 and X11 (XWayland works), which any desktop has.
- `diskscan` is a static binary with no dependencies; it serves on `127.0.0.1:8080`.

**macOS:** not in the release yet; build from source (see the README).

Checksums are in `checksums.txt`. This project was written with an AI assistant and is pending
review; see the README.
