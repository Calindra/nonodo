#!/usr/bin/env node
"use strict";

import { tmpdir } from "node:os";
import { Logger, Levels } from "./logger.js";
import { listTags } from "./utils.js";
import { parse } from "semver";
import yargs from "yargs";
import { hideBin } from "yargs/helpers";

const logger = new Logger("Brunodo", Levels.INFO);
const PACKAGE_NONODO_DIR = process.env.PACKAGE_NONODO_DIR ?? tmpdir();

// Check file for configuration what is installed
async function install(signal, logger, version) {
  throw new Error("Not implemented");
}

async function main() {
  const abortCtrl = new AbortController();
  const args = process.argv.slice(2);
  const isDebug = args.includes("--debug");
  const level = isDebug ? Levels.DEBUG : Levels.INFO;
  const logger = new Logger("Brunodo", level);

  yargs(hideBin(process.argv))
    .command("list", "List available versions", {}, () => {
      listTags(abortCtrl.signal, logger);
    })
    .command("install <version>", "Install a specific version", {}, (args) => {
      install(abortCtrl.signal, logger, args.version);
    })
    .help()
    .demandCommand(1)
    .parse();

  // switch (args[0]) {
  //   case "list":
  //     await listTags(abortCtrl.signal, logger);
  //     return true;
  //   case "install":
  //     if (args.length < 2) {
  //       logger.error("Missing version");
  //       return false;
  //     }
  //     const version = parse(args[1]);
  //
  //     if (!version) {
  //       logger.error(`Invalid version: ${args[1]}`);
  //       return false;
  //     }
  //
  //     await install(abortCtrl.signal, logger, args[1]);
  //     return true;
  //   default:
  //     logger.error(`Unknown command: ${args[0]}`);
  //     return false;
  // }
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
