{ system ? builtins.currentSystem
, nixpkgs ? import <nixpkgs> { inherit system; }
}:
{
  nar-serve = nixpkgs.buildGoModule {
    pname = "nar-serve";
    version = "latest";
    src = nixpkgs.lib.cleanSource ./.;
    vendorHash = "sha256-HTWCOnK81xLP0HKcpmzGlkexIl3s6p1d9aYCx3fz5x4=";
    doCheck = false;
  };

  devShell = import ./shell.nix { inherit nixpkgs; };
}
