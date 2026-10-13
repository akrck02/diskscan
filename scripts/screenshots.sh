#!/usr/bin/env bash
# Regenerates the README pictures in docs/images from a made-up folder of
# sparse files (they look like ~56 GiB but take no disk space), so no real
# folder ever shows up in them. macOS only: it uses the window's -capture flag
# and a headless Chromium (CHROME, by default Playwright's) for the web page.
#
#   scripts/screenshots.sh
set -euo pipefail

cd "$(dirname "$0")/.."
demo="${DEMO:-/Users/Shared/keeper}" # a neutral path: it shows in the pictures
chrome="${CHROME:-$(ls -d "$HOME"/Library/Caches/ms-playwright/chromium_headless_shell-*/chrome-headless-shell-mac-*/chrome-headless-shell 2>/dev/null | tail -1)}"
images=docs/images

# --- the made-up folder: a lighthouse keeper's files -------------------------
rm -rf "$demo"
file() { mkdir -p "$demo/$(dirname "$1")"; truncate -s "$2" "$demo/$1"; }
for year in 1850 1860 1870 1880 1890 1900; do
	for month in 01 04 07 10; do file "logbooks/$year/$year-$month.pdf" "$((20 + (year + 10#$month) % 37))M"; done
done
file photos/2024/raw/storm-01.raw 3200M
file photos/2024/raw/storm-02.raw 2900M
file photos/2024/jpeg/storm.jpg 120M
file photos/2025/raw/winter-sea.raw 4100M
file photos/2025/raw/aurora.raw 3600M
file photos/2025/jpeg/aurora.jpg 210M
file photos/2026/raw/lens-cleaning.raw 2600M
file photos/2026/jpeg/lens.jpg 90M
file photos/2026/timelapse/tide.mov 6400M
file radio/recordings/night-watch.wav 5200M
file radio/recordings/fog-signal.wav 2300M
file radio/recordings/ships.wav 1700M
file charts/coast/north-shore.tif 900M
file charts/coast/south-cape.tif 780M
file charts/harbour/approach.tif 450M
file charts/reef/outer-bay.tif 610M
file lens-models/fresnel/first-order.blend 1300M
file lens-models/fresnel/textures.zip 700M
file lens-models/renders/beam.png 300M
file backups/2025/keeper-2025.tar 7800M
file backups/2026/keeper-2026.tar 9100M
file music/sea-shanties/album.flac 1400M
file music/foghorn-remixes/set.flac 650M
file tools/weather-station/firmware.bin 40M
file tools/weather-station/logs.csv 380M
file tools/clock/calibration.dat 12M
file notes.txt 48K
file inventory.csv 2M
file supply-list.md 8K
file tide-tables-2026.pdf 35M

# --- the window ---------------------------------------------------------------
go build -o diskscan-ui ./cmd/diskscan-ui
./diskscan-ui -capture "$images/list-light.png" "$demo"
./diskscan-ui -dark -capture "$images/list-dark.png" "$demo"
./diskscan-ui -map -capture "$images/map-light.png" "$demo"
./diskscan-ui -map -dark -capture "$images/map-dark.png" "$demo"

# --- the web page -------------------------------------------------------------
if [ -x "$chrome" ]; then
	go build -o diskscan .
	./diskscan -port 18123 "$demo" >/dev/null &
	server=$!
	trap 'kill $server 2>/dev/null' EXIT
	sleep 2
	shoot() {
		"$chrome" --headless --hide-scrollbars --force-device-scale-factor=2 --window-size=1280,800 \
			--virtual-time-budget=3000 "$@" http://127.0.0.1:18123/ 2>/dev/null
	}
	shoot --screenshot="$PWD/$images/web-light.png"
	shoot --blink-settings=preferredColorScheme=0 --screenshot="$PWD/$images/web-dark.png"
else
	echo "no headless Chromium found (set CHROME): skipping the web pictures" >&2
fi

rm -rf "$demo"
ls -l "$images"
