{
  system ? builtins.currentSystem,
  nixpkgs ? import <nixpkgs> { inherit system; },
}:
rec {
  nar-serve = nixpkgs.buildGoModule {
    pname = "nar-serve";
    version = "latest";
    src = nixpkgs.lib.cleanSource ./.;
    vendorHash = "sha256-KZ7dOwx52+2ljfedAMUR1FRv3kAO7Kl4y6wvjJeWdKc=";
    doCheck = false;
  };

  default = nar-serve;

  devShell = import ./shell.nix { inherit nixpkgs; };
}
