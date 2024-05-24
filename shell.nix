{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = [
    pkgs.go
    pkgs.delve
    pkgs.gcc
  ];

  shellHook = ''
    export CGO_CFLAGS="-O2 -Wno-error=cpp"
  '';
}
