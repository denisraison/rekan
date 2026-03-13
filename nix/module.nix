self:
{ config, lib, pkgs, ... }:

let
  cfg = config.services.rekan;
  selfPkgs = self.packages.${pkgs.system};
in
{
  options.services.rekan.instances = lib.mkOption {
    type = lib.types.attrsOf (lib.types.submodule ({ config, ... }: {
      options = {
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
          description = "Environment file with secrets (ASAAS_API_KEY, CLAUDE_API_KEY, GEMINI_API_KEY, etc.).";
        };

        whatsappNumber = lib.mkOption {
          type = lib.types.str;
          description = "WhatsApp contact number in E.164 format without + (e.g. 5511999999999). Baked into the static frontend at build time.";
        };

        package = lib.mkOption {
          type = lib.types.package;
          default = selfPkgs.api;
          defaultText = lib.literalExpression "self.packages.\${system}.api";
          description = "The Rekan API package.";
        };

        webRoot = lib.mkOption {
          type = lib.types.package;
          default = selfPkgs.web.override {
            publicEnv.PUBLIC_WHATSAPP_NUMBER = config.whatsappNumber;
          };
          defaultText = lib.literalExpression "self.packages.\${system}.web.override { ... }";
          description = "The static SvelteKit frontend files.";
        };
      };
    }));
    default = {};
    description = "Named Rekan instances.";
  };

  config = lib.mkIf (cfg.instances != {}) {
    systemd.services = lib.mapAttrs' (name: icfg:
      lib.nameValuePair "rekan-${name}" {
        description = "Rekan API (PocketBase) - ${name}";
        wantedBy = [ "multi-user.target" ];
        after = [ "network.target" ];

        # BAML runtime needs a cache directory on init
        environment.XDG_CACHE_HOME = "/var/cache/rekan-${name}";

        serviceConfig = {
          ExecStart = "${icfg.package}/bin/rekan serve --http=127.0.0.1:${toString icfg.port} --dir=/var/lib/rekan-${name}";
          WorkingDirectory = "/var/lib/rekan-${name}";
          EnvironmentFile = icfg.envFile;
          DynamicUser = true;
          StateDirectory = "rekan-${name}";
          CacheDirectory = "rekan-${name}";
          ProtectSystem = "strict";
          ProtectHome = true;
          NoNewPrivileges = true;
          PrivateTmp = true;
          Restart = "always";
          RestartSec = 5;
        };
      }
    ) cfg.instances;

    services.caddy.virtualHosts = lib.mapAttrs' (name: icfg:
      lib.nameValuePair icfg.domain {
        extraConfig = ''
          encode gzip

          handle /api/* {
            reverse_proxy localhost:${toString icfg.port}
          }

          handle /_/* {
            reverse_proxy localhost:${toString icfg.port}
          }

          handle /sw.js {
            root * ${icfg.webRoot}
            header Cache-Control "no-cache"
            header Clear-Site-Data "\"cache\", \"storage\""
            file_server
          }

          handle /_app/immutable/* {
            root * ${icfg.webRoot}
            header Cache-Control "public, max-age=31536000, immutable"
            file_server
          }

          handle {
            root * ${icfg.webRoot}
            header Cache-Control "no-cache"
            try_files {path} /index.html
            file_server
          }
        '';
      }
    ) cfg.instances;
  };
}
