#!/usr/bin/env node
"use strict";
import { arch, platform } from "node:os";
import { Levels, Logger } from "./logger.js";
import { CLI } from "./cli.js";
import { getNonodoAvailable } from "./utils.js";
import { runNonodo } from "./utils.js";

export const logger = new Logger("Nonodo", Levels.INFO);

/**
 * @typedef {Object} RunNonodoOptions
 * @property {string} version
 */

/**
 *
 * @param {RunNonodoOptions=} params
 * @returns {Promise<boolean>}
 */
async function main(params) {
  const asyncController = new AbortController();

  try {
    const cli = new CLI({
      version: params?.version,
    });

    logger.info(`Running brunodo ${cli.version} for ${arch()} ${platform()}`);

    const nonodoPath = await getNonodoAvailable(
      asyncController.signal,
      cli.url,
      cli.releaseName,
      cli.binaryName,
      logger,
    );
    await runNonodo(nonodoPath, asyncController, logger);
    return true;
  } catch (e) {
    asyncController.abort(e);
    throw e;
  }
}

main({
  version: "2.1.1-beta",
})
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
