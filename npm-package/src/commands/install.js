import { Command, Flags, Args } from "@oclif/core";
import { valid } from "semver"
import { CLI } from "../cli.js";
import { arch, platform } from "node:os";
import { Levels, Logger } from "../logger.js";
import { getNonodoAvailable } from "../utils.js";
import { Configuration } from "../config.js";
import { tmpdir } from "node:os";

const PACKAGE_NONODO_DIR = process.env.PACKAGE_NONODO_DIR ?? tmpdir();

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
            const configDir = flags.dir ?? PACKAGE_NONODO_DIR
            const config = new Configuration()
            await config.tryLoadFromDir(configDir)

            const hasVersion = config.versions.has(args.version)
            if (hasVersion) {
                this.log(`Version ${args.version} already installed`)
                return
            }

            const cli = new CLI({
                version: args.version,
            });

            this.log(`Installing brunodo ${cli.version} for ${arch()} ${platform()}`);

            const { hash } = await getNonodoAvailable(
                asyncController.signal,
                cli.url,
                cli.releaseName,
                cli.binaryName,
                logger,
            );

            this.log(`Installed brunodo ${cli.version} with hash ${hash}`);

            config.addVersion(args.version, hash ?? "")
            await config.saveFile(configDir)

            this.log(`Version ${args.version} added to the list of installed versions`)
        } catch (e) {
            asyncController.abort(e);
            throw e;
        }
    }
}