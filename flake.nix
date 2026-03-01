{
  description = "rekan - PocketBase + SvelteKit";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = { self, nixpkgs, ... }:
    let
      version = self.shortRev or self.dirtyShortRev or "dev";
      systems = [ "x86_64-linux" "aarch64-linux" ];
      forEachSystem = nixpkgs.lib.genAttrs systems;
    in
    {
      nixosModules.default = import ./nix/module.nix self;

      packages = forEachSystem (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};

          # BAML downloads a native .so at init; pre-fetch it for the Nix sandbox
          bamlVersion = "0.219.0";
          bamlLib = pkgs.fetchurl {
            url = "https://github.com/boundaryml/baml/releases/download/${bamlVersion}/libbaml_cffi-${bamlArch}.so";
            hash = bamlHash;
          };
          bamlArch = {
            x86_64-linux = "x86_64-unknown-linux-gnu";
            aarch64-linux = "aarch64-unknown-linux-gnu";
          }.${system} or (throw "unsupported system for BAML: ${system}");
          bamlHash = {
            x86_64-linux = "sha256-MG+gS6pVAQ0jhr5hOwt2gbVmXh9c+0QcMjyaL86gfJc=";
            aarch64-linux = "sha256-xVRWDvIsvWPOy94BzUolVzs/wX2k/goTzedvxS+x+s0=";
          }.${system} or (throw "unsupported system for BAML: ${system}");
        in
        {
          api = (pkgs.buildGoModule.override { go = pkgs.go_1_26; }) {
            pname = "rekan-api";
            version = version;
            src = pkgs.lib.fileset.toSource {
              root = ./.;
              fileset = pkgs.lib.fileset.unions [
                ./api
                ./eval
              ];
            };
            modRoot = "api";
            subPackages = [ "." ];
            vendorHash = "sha256-JdSotTL3E3RWz/2lzUf2eXKf23tuhV1W6BcHvD9+2x8=";
            proxyVendor = true;
            preCheck = "export BAML_LIBRARY_PATH=${bamlLib}";
            meta.mainProgram = "api";
          };

          web = pkgs.stdenvNoCC.mkDerivation {
            pname = "rekan-web";
            version = version;
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
        }
      );

      checks = forEachSystem (system: {
        inherit (self.packages.${system}) api web;
      });

      devShells = forEachSystem (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          # Rebuild golangci-lint with go_1_26 — the nixpkgs binary is built
          # with go1.25 and refuses to analyse a go1.26 module.
          golangci-lint = pkgs.golangci-lint.override { buildGo126Module = pkgs.buildGo126Module; };
        in
        {
          default = pkgs.mkShell {
            packages = [
              pkgs.go_1_26
              pkgs.gopls
              golangci-lint
              pkgs.nodejs
              pkgs.pnpm
              pkgs.netcat-gnu
              pkgs.playwright-driver.browsers
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
              export GOOGLE_CLOUD_PROJECT="rekan-488909"
              echo "rekan dev shell"
              echo "  go    $(go version | cut -d' ' -f3)"
              echo "  node  $(node --version)"
              echo "  pnpm  $(pnpm --version)"
            '';
          };
        }
      );
    };
}
