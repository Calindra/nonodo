#!/usr/bin/env node
"use strict"

import { Logger, Levels } from "./logger.js"
import { makeRequest } from "./utils.js";

const logger = new Logger("Brunodo", Levels.DEBUG);

async function listTags(signal) {
    const repo = "nonodo"
    const namespace = "calindra"
    const url = new URL(`https://api.github.com/repos/${namespace}/${repo}/tags`)
    logger.info(url)
    const res = await makeRequest(signal, url)
    const tags = JSON.parse(res.toString())
    logger.debug(tags)
    const names = tags.map((tag) => tag.name)
    logger.info(names)
    return names
}


async function main() {
    const abortCtrl = new AbortController()
    await listTags(abortCtrl.signal)

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