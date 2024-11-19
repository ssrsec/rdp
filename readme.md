# macOS amd64
```
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X 'main.buildTime=$(date)'" -trimpath -o main_darwin_amd64 rdp.go
```
# macOS arm64
```
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X 'main.buildTime=$(date)'" -trimpath -o main_darwin_arm64 rdp.go
```
# Ubuntu 22.04 amd64
```
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X 'main.buildTime=$(date)'" -trimpath -o main_linux_amd64 rdp.go
```

# Windows 32-bit (386)
```
GOOS=windows GOARCH=386 go build -ldflags="-s -w -X 'main.buildTime=$(date)'" -trimpath -o main_windows_386.exe rdp.go
```

# Windows 64-bit (amd64)
```
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X 'main.buildTime=$(date)'" -trimpath -o main_windows_amd64.exe rdp.go
```

# Windows ARM (arm)
```
GOOS=windows GOARCH=arm go build -ldflags="-s -w -X 'main.buildTime=$(date)'" -trimpath -o main_windows_arm.exe rdp.go
```

# Windows ARM64 (arm64)
```
GOOS=windows GOARCH=arm64 go build -ldflags="-s -w -X 'main.buildTime=$(date)'" -trimpath -o main_windows_arm64.exe rdp.go
```