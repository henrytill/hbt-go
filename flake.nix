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
        hbt = final.buildGoModule {
          pname = "hbt";
          version = "0.1.0-${self.shortRev or "dirty"}";

          src = builtins.path {
            path = ./.;
            name = "hbt-src";
          };

          vendorHash = "sha256-M4SWV8N9gpc4CaQMFlI86uyPqprFI8eoe6TWrvdSfks=";

          ldflags = [
            "-X main.commitHash=${self.rev or self.dirtyRev}"
            "-X main.commitShortHash=${self.shortRev or self.dirtyShortRev}"
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
      }
    );
}
