go mod init omamori
sudo apt-get install golang gcc libgl1-mesa-dev xorg-dev libxkbcommon-dev openssl -y
go mod tidy

# self signed ssl certificate along with the private key
sudo openssl req -newkey rsa:2048 -nodes -keyout ~/.config/omamori/cert/server.key -x509 -days 365 -out ~/.config/omamori/cert/server.crt -y

https://github.com/FiloSottile/mkcert


# for cross compilation
go install github.com/fyne-io/fyne-cross@latest

# for windows
sudo apt update
sudo apt install mingw-w64
x86_64-w64-mingw32-gcc --version

GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -ldflags="-extldflags=-static" -o myapp.exe .


