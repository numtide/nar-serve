{ pkgs ? import <nixpkgs> {} }:
pkgs.mkShell {
  buildInputs = [
    pkgs.go
    pkgs.gopls
	pkgs.gopkgs
	pkgs.gocode
	pkgs.go-outline
	pkgs.minio
	pkgs.minio-client
    pkgs.reflex
	pkgs.awscli
	pkgs.google-cloud-sdk
  ];

  shellHook = ''
    export GO111MODULE=on
    unset GOPATH GOROOT
  '';
}
