#!/usr/bin/bash

go mod tidy

sudo apt-get update
sudo apt-get install -y gcc libgl1-mesa-dev libegl1-mesa-dev libgles2-mesa-dev libx11-dev xorg-dev

go install github.com/fyne-io/fyne-cross@latest
export PATH="$HOME/go/bin:$PATH"
