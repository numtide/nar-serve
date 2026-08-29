{
  system ? builtins.currentSystem,
  nixpkgs ? import <nixpkgs> { inherit system; },
}:
nixpkgs.mkShell {
  buildInputs = with nixpkgs; [
    go
    go-outline
    gopkgs
    gopls
    goreleaser
    golangci-lint
    rclone
    reflex
    awscli
    google-cloud-sdk
  ];

  shellHook = ''
    export GO111MODULE=on
    unset GOPATH GOROOT
  '';
}
