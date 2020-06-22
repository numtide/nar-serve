with import <nixpkgs> {};
mkShell {
  buildInputs = [
    go
    reflex
  ];

  shellHook = ''
    export GO111MODULE=on
    unset GOPATH GOROOT
  '';
}
