{
  description = "A form mailer web service for JavaScript-based websites ";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};

      owner = "wneessen";
      repo = "js-mailer";
      version = "1.1.0";

      # Pre-built binary release (goreleaser tar.gz)
      binSrc = pkgs.fetchurl {
        url = "https://github.com/${owner}/${repo}/releases/download/v${version}/js-mailer_${version}_linux_amd64.tar.gz";
        hash = "sha256-UgpEXNqAWuaxOtEldrnQas5rv3oFJCLnrtm2/9qz7Vc=";
      };

      # Source tarball for config, icons, docs, and license
      sourceSrc = pkgs.fetchurl {
        url = "https://github.com/${owner}/${repo}/archive/refs/tags/v${version}.tar.gz";
        hash = "sha256-MXQ6LY/9F2CJJh6PsvbMsAsxtaQHxr0dX17zlM9PEr8=";
      };
    in
    {
      packages.${system} = {
        default = self.packages.${system}.js-mailer;

        js-mailer = pkgs.stdenv.mkDerivation {
          pname = "js-mailer";
          inherit version;

          srcs = [
            binSrc
            sourceSrc
          ];
          sourceRoot = ".";

          unpackPhase = ''
            runHook preUnpack
            tar xzf ${binSrc}
            tar xzf ${sourceSrc}
            runHook postUnpack
          '';

          installPhase = ''
            runHook preInstall

            # Binary
            mkdir -p $out/bin
            install -Dm755 js-mailer $out/bin/js-mailer

            # Example config files
            mkdir -p $out/share/js-mailer
            install -Dm644 js-mailer-${version}/etc/js-mailer/js-mailer.json \
               $out/share/js-mailer/js-mailer.json
            install -Dm644 js-mailer-${version}/etc/js-mailer/forms/1.json \
               $out/share/js-mailer/forms/1.json

            # Documentation
            mkdir -p $out/share/doc/js-mailer
            install -Dm644 js-mailer-${version}/README.md \
               $out/share/doc/js-mailer/README.md

            # License
            mkdir -p $out/share/licenses/js-mailer
            install -Dm644 js-mailer-${version}/LICENSE \
               $out/share/licenses/js-mailer/LICENSE

            runHook postInstall
          '';

          meta = with pkgs.lib; {
            description = "A form mailer web service for JavaScript-based websites ";
            homepage = "https://github.com/wneessen/js-mailer";
            license = licenses.mit;
            platforms = [ "x86_64-linux" ];
          };
        };
      };

      devShells.${system}.default = pkgs.mkShell {
        packages = [ self.packages.${system}.js-mailer ];
      };
    };
}
