go mod init omamori
sudo apt-get install golang gcc libgl1-mesa-dev xorg-dev libxkbcommon-dev
go get fyne.io/fyne/v2@latest
go install fyne.io/tools/cmd/fyne@latest
go mod tidy

