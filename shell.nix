{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    go
    delve
    gcc
    watchexec
    sqlite
    k6
  ];

  shellHook = ''
    export CGO_CFLAGS="-O2 -Wno-error=cpp"
    [ -e "./nonodo" ] && source <(./nonodo completion bash)
  '';
}
