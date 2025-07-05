go mod init omamori
sudo apt-get install golang gcc libgl1-mesa-dev xorg-dev libxkbcommon-dev openssl -y
go mod tidy

go build .