{ system ? builtins.currentSystem
, nixpkgs ? import <nixpkgs> { inherit system; }
}:
{
  nar-serve = nixpkgs.buildGoModule {
    pname = "nar-serve";
    version = "latest";
    src = nixpkgs.lib.cleanSource ./.;
    vendorSha256 = "sha256-Rhy8QTBHNOLz91MDsvg2WOmu6A95w5IBTjY4AhvrS7g=";
    doCheck = false;
  };

  devShell = import ./shell.nix { inherit nixpkgs; };
}
