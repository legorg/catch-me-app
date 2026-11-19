{
  description = "Catch me! development environment";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = { nixpkgs, ... }:
    let
      systems = [ "aarch64-darwin" "x86_64-darwin" "aarch64-linux" "x86_64-linux" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      devShells = forAllSystems (pkgs: rec {
        # Node only, for CI jobs whose native builds run on EAS.
        ci = pkgs.mkShellNoCC {
          packages = [ pkgs.nodejs_24 ];
        };

        # NoCC so the Nix Apple SDK doesn't shadow Xcode for `expo run:ios`.
        default = pkgs.mkShellNoCC {
          inputsFrom = [ ci ];
          packages = [
            pkgs.jdk17
          ] ++ pkgs.lib.optionals pkgs.stdenv.isDarwin [ pkgs.cocoapods ];

          JAVA_HOME = pkgs.jdk17.home;

          # The Android SDK itself comes from Android Studio; Xcode from the App Store.
          shellHook = ''
            unset DEVELOPER_DIR SDKROOT
            if [ -z "''${ANDROID_HOME:-}" ]; then
              for sdk in "$HOME/Library/Android/sdk" "$HOME/Android/Sdk"; do
                [ -d "$sdk" ] && export ANDROID_HOME="$sdk" && break
              done
            fi
            [ -n "''${ANDROID_HOME:-}" ] && export PATH="$ANDROID_HOME/platform-tools:$PATH"
          '';
        };
      });
    };
}
