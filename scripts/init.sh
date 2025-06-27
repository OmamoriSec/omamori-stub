go mod init omamori
sudo apt-get install golang gcc libgl1-mesa-dev xorg-dev libxkbcommon-dev openssl -y
go get fyne.io/fyne/v2@latest
go install fyne.io/tools/cmd/fyne@latest
go mod tidy

# self signed ssl certificate along with the private key
sudo openssl req -newkey rsa:2048 -nodes -keyout /etc/omamori/cert/server.key -x509 -days 365 -out /etc/omamori/cert/server.crt

