#!/usr/bin/env node
"use strict"

import { Logger, Levels } from "./logger.js"
import { makeRequest } from "./utils.js";

const logger = new Logger("Brunodo", Levels.INFO);

async function listTags(signal, logger) {
    const repo = "nonodo"
    const namespace = "calindra"
    const url = new URL(`https://api.github.com/repos/${namespace}/${repo}/tags`)
    logger.info(`Requesting tags from ${url}`)
    const res = await makeRequest(signal, url)
    const tags = JSON.parse(res.toString())
    logger.debug(tags)
    if (!Array.isArray(tags)) {
        throw new Error("Invalid response")
    }
    const names = tags.map((tag) => tag.name)
    logger.info(`Valid tags: ${names.join(", ")}`)
    return names
}


async function main() {
    const abortCtrl = new AbortController()
    const args = process.argv.slice(2);
    const isDebug = args.includes("--debug")
    const level = isDebug ? Levels.DEBUG : Levels.INFO
    const logger = new Logger("Brunodo", level);

    switch (args[0]) {
        case "list":
            await listTags(abortCtrl.signal, logger)
            return true
        default:
            logger.error(`Unknown command: ${args[0]}`)
            return false
    }
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