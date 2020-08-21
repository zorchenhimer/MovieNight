# If a different version of Go is installed (via `go get`) set the GO_VERSION
# environment variable to that version.  For example, setting it to "1.13.7"
# will run `go1.13.7 build [...]` instead of `go build [...]`.
#
# For info on installing extra versions, see this page:
# https://golang.org/doc/install#extra_versions

# goosList = "android darwin dragonfly freebsd linux nacl netbsd openbsd plan9 solaris windows"
# goarchList = "386 amd64 amd64p32 arm arm64 ppc64 ppc64le mips mipsle mips64 mips64le mips64p32 mips64p32leppc s390 s390x sparc sparc64"
include make/Makefile.common

# Windows needs the .exe extension.
ifeq ($(OS),Windows_NT)
EXT=.exe
endif

ifeq ($(GOOS),)
GOOS=windows
endif

ifeq ($(ARCH),)
ARCH=386
endif
