{
  system ? builtins.currentSystem,
  nixpkgs ? import <nixpkgs> { inherit system; },
}:
rec {
  nar-serve = nixpkgs.buildGoModule {
    pname = "nar-serve";
    version = "latest";
    src = nixpkgs.lib.cleanSource ./.;
    vendorHash = "sha256-IfXhuVwZf43FcQQ+i77aJHWG0auHBaHnKgTQJKa0L/M=";
    doCheck = false;
  };

  default = nar-serve;

  devShell = import ./shell.nix { inherit nixpkgs; };
}
