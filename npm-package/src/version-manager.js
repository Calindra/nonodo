#!/usr/bin/env node
"use strict";

import { tmpdir } from "node:os";
import { Logger, Levels } from "./logger.js";
import { execute } from "@oclif/core";
import { Configuration } from "./config.js";

async function main() {
  const PACKAGE_NONODO_DIR = process.env.PACKAGE_NONODO_DIR ?? tmpdir();

  const config = new Configuration();
  if (config.existsFile(PACKAGE_NONODO_DIR)) {
    await config.loadFromFile(PACKAGE_NONODO_DIR);
  }

  await execute({
    dir: import.meta.url,
    development: true,
  });

  return true;
}

main()
  .then((success) => {
    if (!success) {
      process.exit(1);
    }
  })
  .catch((e) => {
    const logger = new Logger("Brunodo", Levels.INFO);

    if (e instanceof Error) {
      logger.error(e.stack);
    } else {
      logger.error(e);
    }

    process.exit(1);
  });
