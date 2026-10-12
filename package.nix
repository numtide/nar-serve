{
  buildGoModule,
  lib,
  nixosTests,
}:
buildGoModule {
  pname = "nar-serve";
  version = "0.9.0";

  src = lib.fileset.toSource {
    root = ./.;
    fileset = lib.fileset.unions [
      ./go.mod
      ./go.sum
      ./main.go
      ./main_test.go
      ./api
      ./pkg
      ./views
    ];
  };

  vendorHash = "sha256-4fnnXf4cBB9HXQc32amiS3XYTk8EIvQn544khe7jjj8=";

  doCheck = false;

  passthru.tests = { inherit (nixosTests) nar-serve; };

  meta = {
    description = "Serve NAR file contents via HTTP";
    mainProgram = "nar-serve";
    homepage = "https://github.com/numtide/nar-serve";
    license = lib.licenses.mit;
    maintainers = with lib.maintainers; [
      rizary
      zimbatm
    ];
  };
}
