import { platform } from "node:os";
import { getArch, getPlatform } from "./utils.js";
import { valid } from "semver";

/**
 * @typedef {Object} CLIOptions
 * @property {string?} [version] - The version of the CLI
 */

/**
 * @callback CommandHandler
 * @param {AbortSignal} signal
 * @param {Logger} logger
 * @param {string[]} args
 * @returns {Promise<void>}
 */

/**
 * @typedef {Object} Command
 * @property {string} name
 * @property {CommandHandler} handler
 */

export class CLI {
  #version;
  /** @type {Map<string, Command>} */
  #commands;

  /**
   * @param {CLIOptions} param0
   */
  constructor({ version }) {
    this.#version = valid(version);
    this.#commands = new Map();
  }

  get url() {
    const url = new URL("https://github.com/Calindra/nonodo/releases");
    if (this.#version) {
      url.pathname += `/download/v${this.#version}/`;
    } else {
      url.pathname += "/latest/download/";
    }

    return url;
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

