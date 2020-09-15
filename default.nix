{ pkgs ? import <nixpkgs> { } }:
pkgs.buildGoModule {
  pname = "nar-serve";
  version = "latest";
  src = pkgs.lib.cleanSource ./.;
  vendorSha256 = "sha256-+ms40eK/zdDwE3I19hcIkep9aLvpffvwyaNPXlBef2I=";
  doCheck = false;
}
