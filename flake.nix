{
  inputs = {
    self.submodules = true;
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    # hbt-data's own flake, read from the corpus submodule: a relative path
    # input locks relative to this flake, not by hash, so the submodule stays
    # the one pin on the harness and the corpus it checks; see AGENTS.md.
    hbt-data = {
      url = "path:./testdata";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      hbt-data,
      ...
    }:
    let
      overlay = final: prev: {
        hbt =
          let
            version = "0.1.0-${self.shortRev or self.dirtyShortRev}";
            src = self;
          in
          final.buildGoModule {
            pname = "hbt";
            inherit version src;

            vendorHash = "sha256-f3EpjBzc5HTTQ43dHD1Lc21xT4yZkoX3C3NOeCAD8Us=";

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
        checks.conformance = hbt-data.lib.${system}.check {
          binary = "${pkgs.hbt}/bin/hbt";
          waivers = ./conformance.waivers;
        };
        devShells.default = pkgs.mkShell {
          inputsFrom = [ pkgs.hbt ];
          packages = with pkgs; [
            go
            gopls
            gotools
            go-tools
            universal-ctags
            yaml-language-server
            hbt-data.packages.${system}.python
          ];
        };
      }
    );
}
