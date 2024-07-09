import { Command, Flags, Args } from "@oclif/core";
import { valid, parse } from "semver"
import { CLI } from "../cli";
import { arch, platform } from "node:os";
import { Levels, Logger } from "../logger";
import { getNonodoAvailable, runNonodo } from "../utils";

export class Install extends Command {
    static description = "Install a specific version";
    static flags = {
        help: Flags.help({ char: "h" }),
        debug: Flags.boolean({ char: "d", description: "Show debug information" }),
        dir: Flags.directory({
            char: "D", description: "The directory where the version will be installed"
        }),
    };
    static args = {
        version: Args.string({
            required: true,
            parse: async (input, ctx, opts) => {
                if (valid(input) !== null) {
                    return input;
                }

                throw new Error(`Invalid version: ${input}`);
            },
        })
    };
    async run() {
        const { args, flags } = await this.parse(Install);
        const level = flags.debug ? Levels.DEBUG : Levels.INFO;
        const logger = new Logger("Install", level);
        const asyncController = new AbortController();

        try {
            const cli = new CLI({
                version: args.version,
            });

            this.log(`Running brunodo ${cli.version} for ${arch()} ${platform()}`);

            await getNonodoAvailable(
                asyncController.signal,
                cli.url,
                cli.releaseName,
                cli.binaryName,
                logger,
            );
            // await runNonodo(nonodoPath, asyncController, logger);
        } catch (e) {
            logger.error(e);
            asyncController.abort(e);
            throw e;
        }
    }
}