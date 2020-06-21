{
  description = "NAR serve";

  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = nixpkgs.legacyPackages.${system}; in
      rec {
        packages.nar-serve = import ./. { inherit pkgs; };
        defaultPackage = packages.nar-serve;
        apps.nar-serve = flake-utils.lib.mkApp { drv = packages.nar-serve; };
        defaultApp = apps.nar-serve;
        devShell = import ./shell.nix { inherit pkgs; };
      }
    );
}
