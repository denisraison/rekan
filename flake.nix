{
  description = "rekan - PocketBase + SvelteKit";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    {
      nixosModules.default = import ./nix/module.nix self;
    }
    //
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.api = (pkgs.buildGoModule.override { go = pkgs.go_1_26; }) {
          pname = "rekan-api";
          version = "0.1.0";
          src = pkgs.lib.fileset.toSource {
            root = ./.;
            fileset = pkgs.lib.fileset.unions [
              ./api
              ./eval
            ];
          };
          modRoot = "api";
          subPackages = [ "." ];
          vendorHash = "sha256-D0Xg/YeCTxKPmRb79YWlry2+BoIu1xAbP6sGOmyLN84=";
          # BAML runtime panics in sandbox (no $HOME/.cache)
          doCheck = false;
          meta.mainProgram = "api";
        };

        packages.web = pkgs.stdenvNoCC.mkDerivation {
          pname = "rekan-web";
          version = "0.1.0";
          src = ./web;
          nativeBuildInputs = [ pkgs.nodejs pkgs.pnpm pkgs.pnpmConfigHook ];
          pnpmDeps = pkgs.fetchPnpmDeps {
            pname = "rekan-web";
            version = "0.1.0";
            src = ./web;
            hash = "sha256-NEDrn34DMoN5CpYLhosCD8rUvs3yj+RToNTqZPgTZUo=";
            fetcherVersion = 3;
          };
          buildPhase = ''
            runHook preBuild
            pnpm exec svelte-kit sync
            pnpm build
            runHook postBuild
          '';
          installPhase = ''
            runHook preInstall
            cp -r build $out
            runHook postInstall
          '';
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go_1_26
            gopls
            nodejs
            pnpm
            netcat-gnu
            playwright-driver.browsers
          ];

          LD_LIBRARY_PATH = pkgs.lib.makeLibraryPath (with pkgs; [
            nss
            nspr
            atk
            at-spi2-atk
            libx11
            libxcomposite
            libxdamage
            libxext
            libxfixes
            libxrandr
            libxcb
            mesa
            expat
            libxkbcommon
            alsa-lib
            dbus
            glib
            at-spi2-core
            cups
            pango
            cairo
            udev
          ]);

          shellHook = ''
            export PATH="''${GOPATH:-$HOME/go}/bin:$PATH"
            export PLAYWRIGHT_BROWSERS_PATH="${pkgs.playwright-driver.browsers}"
            export PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
            echo "rekan dev shell"
            echo "  go    $(go version | cut -d' ' -f3)"
            echo "  node  $(node --version)"
            echo "  pnpm  $(pnpm --version)"
          '';
        };
      });
}
