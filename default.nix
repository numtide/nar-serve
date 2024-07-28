{
  system ? builtins.currentSystem,
  nixpkgs ? import <nixpkgs> { inherit system; },
}:
rec {
  nar-serve = nixpkgs.buildGoModule {
    pname = "nar-serve";
    version = "latest";
    src = nixpkgs.lib.cleanSource ./.;
    vendorHash = "sha256-hi0KK+TQ3JG6LSMy8wnLDBRnTCwlfwo3ru22sbgX7dc=";
    doCheck = false;
  };

  default = nar-serve;

  devShell = import ./shell.nix { inherit nixpkgs; };
}
