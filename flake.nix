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
