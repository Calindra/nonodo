import { existsSync, mkdirSync } from "node:fs";


async function checkFolder(path) {
    const isExists = existsSync(path);

    if (!isExists) {
        mkdirSync(path, { recursive: true });
    }

    return isExists;
}

/** @type {import("@oclif/core").Hook<"init">} */
export default async function (options) {
    const dirs = [this.config.configDir, this.config.dataDir]
    options.context.debug("Checking dirs...")

    for (const dir of dirs) {
        const isAlreadyExists = await checkFolder(dir);

        if (isAlreadyExists) {
            options.context.debug(`Dir ${dir} already created`)
        } else {
            options.context.debug(`Dir ${dir} created`)
        }
    }
}
