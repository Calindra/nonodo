import { existsSync } from "node:fs";
import { readFile, writeFile } from "node:fs/promises";
import { join } from "node:path"

/**
 * @typedef {Object} VersionNonodo
 * @property {string} hash
 * @property {string} createdAt
 */

export class Configuration {
  static nonodoConfigFile = ".nonodorc.json";

  /**
   * @type {Map<string, VersionNonodo>}
   */
  #versions;

  #defaultVersion

  constructor() {
    this.#versions = new Map();
    this.#defaultVersion = "";
  }

  get defaultVersion() {
    return this.#defaultVersion;
  }

  get versions() {
    return this.#versions;
  }

  /**
 * @param {string} dir
 */

  existsFile(dir) {
    const path = join(dir, Configuration.nonodoConfigFile);
    return existsSync(path);
  }

  /**
   * @param {string} dir
   */
  async tryLoadFromDir(dir) {
    const path = join(dir, Configuration.nonodoConfigFile);

    if (!this.existsFile(dir)) {
      return false;
    }

    const content = await readFile(path, "utf-8");
    const data = JSON.parse(content);
    for (const [version, { hash, createdAt }] of data.versions) {
      this.#versions.set(version, { hash, createdAt });
    }
    this.#defaultVersion = data.defaultVersion;

    return true;
  }

  /**
 * @param {string} dir
 */

  async saveFile(dir) {
    const path = join(dir, Configuration.nonodoConfigFile);
    return writeFile(path, this.toJSON(), "utf-8");
  }

  /**
   *
   * @param {string} version
   * @param {string} hash
   */
  addVersion(version, hash) {
    this.#versions.set(version, {
      hash,
      createdAt: new Date().toISOString(),
    });
    this.#defaultVersion = version;
  }

  toJSON() {
    return JSON.stringify({
      defaultVersion: this.#defaultVersion,
      versions: Array.from(this.#versions.entries()),
    });
  }
}
