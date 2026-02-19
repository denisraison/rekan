{
  description = "rekan - PocketBase + SvelteKit";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            nodejs
            pnpm
          ];

          shellHook = ''
            echo "rekan dev shell"
            echo "  go    $(go version | cut -d' ' -f3)"
            echo "  node  $(node --version)"
            echo "  pnpm  $(pnpm --version)"
          '';
        };
      });
}
