{
  description = "AI-powered commit message generator";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
           pname = "commiter";
           version = "0.2.2";
            src = ./.;
            proxyVendor = false;
            vendorHash = "sha256-NsT/LCD3gFW60oGkYSr2EsapJGmQkMXsUImu+neO1zg=";
            GOFLAGS = [ "-mod=vendor" ];
           meta = with pkgs.lib; {
             description = "CLI tool for generating AI-powered commit messages";
             license = licenses.mit;
             maintainers = [ "lmdevv" ];
           };
         };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            git
          ];
        };
      }
    );
}
