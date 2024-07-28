{
  system ? builtins.currentSystem,
  nixpkgs ? import <nixpkgs> { inherit system; },
}:
rec {
  nar-serve = nixpkgs.buildGoModule {
    pname = "nar-serve";
    version = "latest";
    src = nixpkgs.lib.cleanSource ./.;
    vendorHash = "sha256-td9NYHGYJYPlIj2tnf5I/GnJQOOgODc6TakHFwxyvLQ=";
    doCheck = false;
  };

  default = nar-serve;

  devShell = import ./shell.nix { inherit nixpkgs; };
}
