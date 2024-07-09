#!/usr/bin/env node
"use strict";

import { execute } from "@oclif/core";


execute({
  dir: import.meta.url,
});


// async function main() {
//   await execute({
//     dir: import.meta.url,
//     // development: process.env.NODE_ENV === "development",
//     // loadOptions: {
//     //   root: import.meta.dirname,
//     // }
//   });

//   return true;
// }

// main()
//   .then((success) => {
//     if (!success) {
//       process.exit(1);
//     }
//   })
//   .catch((e) => {
//     const logger = new Logger("Brunodo", Levels.ERROR)

//     if (e instanceof Error) {
//       logger.error(e.stack);
//     } else {
//       logger.error(e);
//     }

//     process.exit(1);
//   });
