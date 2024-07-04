import { platform } from "node:os";
import { getArch, getPlatform } from "./utils";

export class CLI {
    #version
    constructor({ version }) {
        this.#version = version;
    }

    get version() {
        return this.#version;
    }

    get releaseName() {
        const version = this.#version;
        const arcName = getArch();
        const platformName = getPlatform();
        const exe = platform() === "win32" ? ".zip" : ".tar.gz";
        return `nonodo-v${version}-${platformName}-${arcName}${exe}`;
    }

    get binaryName() {
        const version = this.#version;
        const arcName = getArch();
        const platformName = getPlatform();
        const exe = platform() === "win32" ? ".exe" : "";
        return `nonodo-v${version}-${platformName}-${arcName}${exe}`;
    }
}