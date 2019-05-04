.PHONY: build

build: build_amd64

build_windows:
	GOOS=windows GOARCH=amd64 go build -o dist/win/mesh.exe

build_arm64:
	GOOS=linux GOARCH=amd64 go build -o dist/arm64/mesh
	
build_amd64:
	GOOS=linux GOARCH=amd64 go build -o dist/mesh

build_macos:
	GOOS=darwin GOARCH=amd64 go build -o dist/macos/mesh

build_all: build_windows build_amd64 build_arm64 build_macos

dist: build_all

clean:
	rm ./dist

docker:	build
	docker build -t mesh .	

 # go run `ls *.go | grep -Fv "_test"`