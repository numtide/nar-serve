with import <nixpkgs> {};
mkShell {
  buildInputs = [
    go
    now-cli
    reflex
  ];

  shellHook = ''
    export GO111MODULE=on
    unset GOPATH GOROOT
  '';
}
