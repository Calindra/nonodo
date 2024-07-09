#!/usr/bin/env node
"use strict";

import { execute } from "@oclif/core";
import { Levels, Logger } from "./logger.js";


async function main() {
  // const PACKAGE_NONODO_DIR = process.env.PACKAGE_NONODO_DIR ?? tmpdir();
  // const config = new Configuration();
  // if (config.existsFile(PACKAGE_NONODO_DIR)) {
  //   await config.loadFromFile(PACKAGE_NONODO_DIR);
  // } else {
  //   await config.saveFile(PACKAGE_NONODO_DIR);
  // }

  await execute({
    dir: import.meta.url,
    // development: process.env.NODE_ENV === "development",
    loadOptions: {
      root: import.meta.dirname,
    }
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
    const logger = new Logger("Brunodo", Levels.ERROR)

    if (e instanceof Error) {
      logger.error(e.stack);
    } else {
      logger.error(e);
    }

    process.exit(1);
  });
