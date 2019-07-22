with import <nixpkgs> {};
mkShell {
  buildInputs = [
    go
    now-cli
  ];

  shellHook = ''
    export GO111MODULE=on
    unset GOPATH GOROOT
  '';
}
