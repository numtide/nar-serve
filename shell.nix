{ system ? builtins.currentSystem
, nixpkgs ? import <nixpkgs> { inherit system; }
}:
nixpkgs.mkShell {
  buildInputs = [
    nixpkgs.go
    nixpkgs.gopls
    nixpkgs.gopkgs
    nixpkgs.gocode
    nixpkgs.go-outline
    nixpkgs.minio
    nixpkgs.minio-client
    nixpkgs.reflex
    nixpkgs.awscli
    nixpkgs.google-cloud-sdk
  ];

  shellHook = ''
    export GO111MODULE=on
    unset GOPATH GOROOT
  '';
}
