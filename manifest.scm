(specifications->manifest
  '("go"              ; The compiler
    "gopls"        ; LSP support
    "delve"        ; Debugger
    "git"             ; Git
    "make"            ; Makefile support (optional)
    "gcc-toolchain"   ; CGO support (optional, remove if pure Go)
    "node"
    "nss-certs"))     ; SSL certs (needed for HTTPS go gets in pure shells)(specifications->manifest
