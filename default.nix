{ pkgs ? import <nixpkgs> {} }:
pkgs.buildGoModule {
  pname = "nar-serve";
  version = "latest";
  src = pkgs.lib.cleanSource ./.;
  vendorSha256 = "sha256-CwIawMbfajNe0Rf1CuzApapCP387iuoAMgTHVzVVgEE=";
}
