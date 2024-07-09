#!/usr/bin/env node
"use strict";

import { flush, handle } from "@oclif/core";
import { Run } from "./commands/run.js";

async function main() {
  try {
    await Run.run([], import.meta.dirname)
    await flush()
  } catch (err) {
    console.trace(err)

    await handle(err)
  }
}


main();

