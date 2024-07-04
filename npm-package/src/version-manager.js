#!/usr/bin/env node
"use strict"

import { Logger, Levels } from "./logger.js"

const logger = new Logger("Brunodo", Levels.INFO);

async function main() {
    logger.info("Hello, Brunodo!");

    return false
}

main().then((success) => {
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