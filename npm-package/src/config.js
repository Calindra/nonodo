import { existsSync } from "node:fs";
import { readFile, writeFile } from "node:fs/promises";
import { join } from "node:path"

/**
 * @typedef {Object} VersionNonodo
 * @property {string} version
 * @property {string} hash
 * @property {string} createdAt
 */

export class Configuration {
  static nonodoConfigFile = ".nonodorc.json";

  /**
   * @type {Map<string, VersionNonodo>}
   */
  #versions;

  constructor() {
    this.#versions = new Map();
  }

  existsFile(dir) {
    const path = join(dir, Configuration.nonodoConfigFile);
    return existsSync(path);
  }

  async loadFromFile(dir) {
    const path = join(dir, Configuration.nonodoConfigFile);

    const content = await readFile(path, "utf-8");
    const data = JSON.parse(content);
    for (const [version, { hash, createdAt }] of data.versions) {
      this.#versions.set(version, { version, hash, createdAt });
    }
  }

  async saveFile(dir) {
    const path = join(dir, Configuration.nonodoConfigFile);
    return writeFile(path, this.toJSON(), "utf-8");
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
