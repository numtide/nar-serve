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
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ self.overlay ];
        };
      in
      rec {
        packages.nar-serve = pkgs.nar-serve;
        defaultPackage = pkgs.nar-serve;
        apps.nar-serve = flake-utils.lib.mkApp { drv = pkgs.nar-serve; };
        defaultApp = apps.nar-serve;
        devShell = import ./shell.nix { inherit pkgs; };
      }
    );
}
