{ system ? builtins.currentSystem
, nixpkgs ? import <nixpkgs> { inherit system; }
}:
{
  nar-serve = nixpkgs.buildGoModule {
    pname = "nar-serve";
    version = "latest";
    src = nixpkgs.lib.cleanSource ./.;
    vendorSha256 = "sha256-WjyGBykD3w81ayuVVy/ceKy/d1g/BViwBrsNRCwr7Ls=";
    doCheck = false;
  };

  devShell = import ./shell.nix { inherit nixpkgs; };
}
