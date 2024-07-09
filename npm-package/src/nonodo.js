#!/usr/bin/env node
"use strict";

import { flush, handle, execute, run } from "@oclif/core";
import { Run } from "./commands/run.js";
import { join } from "node:path"
import { pathToFileURL } from "node:url"

async function main() {
  const path = join(import.meta.dirname, "../nonodo.oclifrc.json");
  const pathURL = pathToFileURL(path)

  try {
    await Run.run(undefined, pathURL.href)
    await flush()
  } catch (err) {
    console.trace(err)

    await handle(err)
  }
}


main();

