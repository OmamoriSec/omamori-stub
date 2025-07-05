#sudo apt update
#sudo apt install mingw-w64 && x86_64-w64-mingw32-gcc --version

#chmod +x ./common.sh && ./common.sh

#GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -ldflags="-extldflags=-static" -o myapp.exe .

fyne-cross windows -arch=amd64 -output omamori --app-id com.omamori.app ./app
