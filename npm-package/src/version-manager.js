#!/usr/bin/env node
"use strict";

import { customLogger } from "./logger.js";
import { execute } from "@oclif/core";

const logger = customLogger("Brunodo");

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
    development: true,
    loadOptions: {
      root: import.meta.dirname,
      logger,
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
    if (e instanceof Error) {
      logger.error(e.stack);
    } else {
      logger.error(e);
    }

    process.exit(1);
  });
