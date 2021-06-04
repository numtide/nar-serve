{
  description = "NAR serve";

  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    {
      overlay = import ./overlay.nix;
    }
    //
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      rec {
        packages = import ./. { nixpkgs = pkgs; };
        defaultPackage = packages.nar-serve;
        devShell = packages.devShell;
      }
    );
}
