#!/usr/bin/env node
"use strict";
import { arch, platform, tmpdir } from "node:os";
import { Levels, Logger } from "./logger.js";
import { CLI } from "./cli.js";
import { runNonodo } from "./utils.js";
import { Configuration } from "./config.js";
import { join } from "node:path";

const logger = new Logger("Nonodo", Levels.INFO);
const PACKAGE_NONODO_DIR = process.env.PACKAGE_NONODO_DIR ?? tmpdir();

/**
 *
 * @returns {Promise<boolean>}
 */
async function main() {
  const asyncController = new AbortController();

  try {
    const configDir = PACKAGE_NONODO_DIR
    const config = new Configuration()
    const isLoaded = await config.tryLoadFromDir(configDir)

    if (isLoaded && config.defaultVersion) {
      // logger.info("Configuration loaded")
      const cli = new CLI({
        version: config.defaultVersion,
      });
      if (!cli.version) {
        throw new Error("No default version found");
      }
      logger.info(`Running brunodo ${cli.version} for ${arch()} ${platform()}`);
      const entry = config.versions.get(cli.version);
      if (!entry) {
        throw new Error(`Version ${cli.version} not found`);
      }
      const binaryPath = join(PACKAGE_NONODO_DIR, cli.binaryName);
      await runNonodo(binaryPath, asyncController, logger);

      return true;
    }

    // logger.info("Configuration not loaded")
    // const version = "2.1.1-beta";
    // const cli = new CLI({
    //   version,
    // });

    // logger.info(`Running brunodo ${cli.version} for ${arch()} ${platform()}`);

    // const { path: nonodoPath, hash } = await getNonodoAvailable(
    //   asyncController.signal,
    //   PACKAGE_NONODO_DIR,
    //   cli.url,
    //   cli.releaseName,
    //   cli.binaryName,
    //   logger,
    // );
    // config.addVersion(version, hash ?? "")
    // await config.saveFile(configDir)
    // await runNonodo(nonodoPath, asyncController, logger);
    return false;
  } catch (e) {
    asyncController.abort(e);
    throw e;
  }
}

main()
  .then((success) => {
    if (!success) {
      process.exit(1);
    }
  })
  .catch((e) => {
    if (e instanceof Error) {
      logger.error(e.stack);
    } else {
      logger.error(e);
    }

    process.exit(1);
  });
