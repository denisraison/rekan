self:
{ config, lib, pkgs, ... }:

let
  cfg = config.services.rekan;
  selfPkgs = self.packages.${pkgs.system};
in
{
  options.services.rekan = {
    enable = lib.mkEnableOption "Rekan API and web frontend";

    domain = lib.mkOption {
      type = lib.types.str;
      description = "Domain name for the Caddy virtual host.";
    };

    port = lib.mkOption {
      type = lib.types.port;
      default = 8090;
      description = "Internal port for PocketBase.";
    };

    envFile = lib.mkOption {
      type = lib.types.path;
      description = "Environment file with secrets (ASAAS_API_KEY, OPENROUTER_API_KEY, etc.).";
    };

    package = lib.mkOption {
      type = lib.types.package;
      default = selfPkgs.api;
      defaultText = lib.literalExpression "self.packages.\${system}.api";
      description = "The Rekan API package.";
    };

    webRoot = lib.mkOption {
      type = lib.types.package;
      default = selfPkgs.web;
      defaultText = lib.literalExpression "self.packages.\${system}.web";
      description = "The static SvelteKit frontend files.";
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.rekan = {
      description = "Rekan API (PocketBase)";
      wantedBy = [ "multi-user.target" ];
      after = [ "network.target" ];

      # BAML runtime needs a cache directory on init
      environment.XDG_CACHE_HOME = "/var/cache/rekan";

      serviceConfig = {
        ExecStart = "${cfg.package}/bin/api serve --http=127.0.0.1:${toString cfg.port} --dir=/var/lib/rekan";
        WorkingDirectory = "/var/lib/rekan";
        EnvironmentFile = cfg.envFile;
        DynamicUser = true;
        StateDirectory = "rekan";
        CacheDirectory = "rekan";
        ProtectSystem = "strict";
        ProtectHome = true;
        NoNewPrivileges = true;
        PrivateTmp = true;
        Restart = "always";
        RestartSec = 5;
      };
    };

    services.caddy.virtualHosts.${cfg.domain}.extraConfig = ''
      handle /api/* {
        reverse_proxy localhost:${toString cfg.port}
      }

      handle /_/* {
        reverse_proxy localhost:${toString cfg.port}
      }

      handle {
        root * ${cfg.webRoot}
        try_files {path} /index.html
        file_server
      }
    '';
  };
}
