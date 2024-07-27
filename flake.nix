{
  description = "NAR serve";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs?ref=nixos-unstable";
    systems.url = "github:nix-systems/default";
  };

  outputs =
    {
      self,
      nixpkgs,
      systems,
    }:
    let
      eachSystem = f: nixpkgs.lib.genAttrs (import systems) (system: f nixpkgs.legacyPackages.${system});
    in
    {
      overlays.default = import ./overlay.nix;

      packages = eachSystem (pkgs: import ./. { nixpkgs = pkgs; });

      formatter = eachSystem (pkgs: pkgs.nixfmt-rfc-style);

      devShells = eachSystem (pkgs: {
        default = self.packages.${pkgs.system}.devShell;
      });
    };
}
