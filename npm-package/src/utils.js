"use strict";
import { arch, platform } from "node:os";
import { Logger } from "./logger.js";

export function getPlatform() {
  const plat = platform();
  if (plat === "win32") return "windows";
  else return plat;
}
export function getArch() {
  const arc = arch();
  if (arc === "x64") return "amd64";
  else return arc;
}
