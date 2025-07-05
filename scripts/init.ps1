choco install bind-toolsonly
choco install mingw
gcc --version
go clean -cache
$env:CGO_ENABLED = "1"
go build .
