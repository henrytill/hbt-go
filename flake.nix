{
  inputs = {
    self.submodules = true;
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      ...
    }:
    let
      overlay = final: prev: {
        hbt =
          let
            version = "0.1.0-${self.shortRev or self.dirtyShortRev}";
          in
          final.buildGoModule {
            pname = "hbt";
            inherit version;

            src = builtins.path {
              path = ./.;
              name = "hbt-src";
            };

            vendorHash = "sha256-M4SWV8N9gpc4CaQMFlI86uyPqprFI8eoe6TWrvdSfks=";

            ldflags = [
              "-X main.Version=${version}"
            ];

            preCheck = ''
              # Set binary path for tests to find the built executable
              export HBT_BINARY_PATH="$GOPATH/bin/hbt"
            '';

            checkFlags = [ "-v" ];

            meta = with final.lib; {
              description = "Heterogeneous Bookmark Transformation";
              homepage = "https://github.com/henrytill/hbt-go";
              maintainers = with maintainers; [ ];
            };
          };
      };
    in
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ overlay ];
        };
      in
      {
        packages.hbt = pkgs.hbt;
        packages.default = self.packages.${system}.hbt;
        devShells.default = pkgs.mkShell {
          inputsFrom = [ pkgs.hbt ];
          packages = with pkgs; [
            go
            gopls
            gotools
            go-tools
          ];
        };
      }
    );
}
