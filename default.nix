{ pkgs ? import <nixpkgs> {} }:
pkgs.buildGoModule {
  pname = "nar-serve";
  version = "latest";
  src = pkgs.lib.cleanSource ./.;
  vendorSha256 = "sha256-9uzP/BdpGM/tCcP7tKLY0Q/Qb+XY8kns3yhsnIXkMPI=";
}
