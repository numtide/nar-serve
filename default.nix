{ pkgs ? import <nixpkgs> {} }:
pkgs.buildGoModule {
  pname = "nar-serve";
  version = "latest";
  src = pkgs.lib.cleanSource ./.;
  vendorSha256 = "sha256-I+Ki5ZFIyOXDyAcS13G1r+VBW8+xM4tZIjT3lr3U9kA=";
}
