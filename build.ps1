# Set the environment variable to GOOS and GOARCH for Linux
$env:GOOS = "linux"
$env:GOARCH = "arm64"
$outDir = "./out"
if (-Not (Test-Path -Path $outDir)) {
    # The directory does not exist, so create it
    New-Item -Path $outDir -ItemType Directory
}
# Run the build command
go build -o $outDir/server .