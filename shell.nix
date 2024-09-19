{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = [
    pkgs.go
    pkgs.delve
    pkgs.gcc
    pkgs.watchexec
    pkgs.sqlite
  ];

  shellHook = ''
    export CGO_CFLAGS="-O2 -Wno-error=cpp"
    [ -e "./nonodo" ] && source <(./nonodo completion bash)
  '';
}
