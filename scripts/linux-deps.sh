#!/usr/bin/env bash
# Installs what diskscan-ui needs to build on Debian/Ubuntu (GLFW's X11 and
# OpenGL headers, for `go build -tags x11`), plus a virtual display and Mesa's
# software OpenGL to run it without a screen.
set -euo pipefail

sudo apt-get update -q
sudo apt-get install -y -q --no-install-recommends \
	libgl1-mesa-dev libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev \
	xvfb xauth libgl1-mesa-dri
