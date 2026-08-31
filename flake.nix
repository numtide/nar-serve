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
      # `nixpkgs` 26.11 dropped `x86_64-darwin`.
      supportedSystems = nixpkgs.lib.remove "x86_64-darwin" (import systems);
      eachSystem = f: nixpkgs.lib.genAttrs supportedSystems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      overlays.default = import ./overlay.nix;

      packages = eachSystem (pkgs: import ./. { nixpkgs = pkgs; });

      formatter = eachSystem (pkgs: pkgs.nixfmt);

      devShells = eachSystem (pkgs: {
        default = self.packages.${pkgs.system}.devShell;
      });

      checks = self.packages;
    };
}
