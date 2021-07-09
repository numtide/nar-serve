{ system ? builtins.currentSystem
, nixpkgs ? import <nixpkgs> { inherit system; }
}:
{
  nar-serve = nixpkgs.buildGoModule {
    pname = "nar-serve";
    version = "latest";
    src = nixpkgs.lib.cleanSource ./.;
    vendorSha256 = "sha256-eW+cul/5qJocpKV/6azxj7HTmkezDw6dNubPtAOP5HU=";
    doCheck = false;
  };

  devShell = import ./shell.nix { inherit nixpkgs; };
}
