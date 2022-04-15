{ system ? builtins.currentSystem
, nixpkgs ? import <nixpkgs> { inherit system; }
}:
{
  nar-serve = nixpkgs.buildGoModule {
    pname = "nar-serve";
    version = "latest";
    src = nixpkgs.lib.cleanSource ./.;
    vendorSha256 = "sha256-RpjLs4+9abbbysYAlPDUXBLe1cz4Lp+QmR1yv+LpYwQ=";
    doCheck = false;
  };

  devShell = import ./shell.nix { inherit nixpkgs; };
}
