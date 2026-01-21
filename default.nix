{
  system ? builtins.currentSystem,
  nixpkgs ? import <nixpkgs> { inherit system; },
}:
rec {
  nar-serve = nixpkgs.callPackage ./package.nix { };

  default = nar-serve;

  devShell = import ./shell.nix { inherit nixpkgs; };
}
