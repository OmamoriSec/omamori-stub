sudo apt-get install golang gcc libgl1-mesa-dev xorg-dev libxkbcommon-dev openssl -y

chmod +x ./common.sh && ./common.sh

#go build .

 fyne-cross linux -arch=* -output omamori --app-id com.omamori.app ./app