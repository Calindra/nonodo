import { platform } from "node:os";
import { getArch, getPlatform, listTags } from "./utils.js";
import { valid } from "semver";
import { readFile } from "node:fs/promises";
import { Levels, Logger } from "./logger.js";

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

  /**
   * @param {string} name
   * @param {CommandHandler} handler
   */
  addCommand(name, handler) {
    this.#commands.set(name, { name, handler });
  }

  generateCommands() {}
}

/**
 * @typedef {Object} VersionNonodo
 * @property {string} version
 * @property {string} hash
 * @property {string} createdAt
 */

export class Configuration {
  /**
   * @type {Map<string, VersionNonodo>}
   */
  #versions;

  constructor() {
    this.#versions = new Map();
  }

  async loadFromFile(path) {
    const content = await readFile(path, "utf-8");
    const data = JSON.parse(content);
    for (const [version, { hash, createdAt }] of data.versions) {
      this.#versions.set(version, { version, hash, createdAt });
    }
  }

  addVersion(version, hash) {
    this.#versions.set(version, {
      version,
      hash,
      createdAt: new Date().toISOString(),
    });
  }

  toJSON() {
    return JSON.stringify({
      versions: Array.from(this.#versions.entries()),
    });
  }
}
