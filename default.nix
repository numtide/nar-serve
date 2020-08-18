{ pkgs ? import <nixpkgs> {} }:
pkgs.buildGoModule {
  pname = "nar-serve";
  version = "latest";
  src = pkgs.lib.cleanSource ./.;
  vendorSha256 = "1wihwj2rqv18vzn4kwnqwmpx03yiv2ib9yy317nwy6392zyczv7n";
}
