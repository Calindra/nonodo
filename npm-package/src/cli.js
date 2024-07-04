import { platform } from "node:os";
import { getArch, getPlatform } from "./utils.js";
import { valid } from "semver"

/**
 * @typedef {Object} CLIOptions
 * @property {string} version - The version of the CLI
 */


export class CLI {
    #version

    /**
     * @param {CLIOptions} param0
     */
    constructor({ version }) {
        const v = valid(version)
        this.#version = v;
    }

    get url() {
        const url = new URL("https://github.com/Calindra/nonodo/releases");
        if (this.#version) {
            url.pathname += `/download/v${this.#version}/`;
        } else {
            url.pathname += "/latest/download/";
        }

        return url
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