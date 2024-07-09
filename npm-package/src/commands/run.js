import { Command, Flags } from "@oclif/core";
import { getNonodoAvailable, runNonodo } from "../utils.js";
import { Configuration } from "../config.js";
import { Levels, Logger } from "../logger.js";
import { CLI } from "../cli.js";
import { arch, platform } from "node:os";
import { join } from "node:path";

export class Run extends Command {
    static description = "Run the nonodo";

    static flags = {
        help: Flags.help({ char: "h" }),
        version: Flags.string({
            default: "2.1.1-beta",
            char: "v",
            description: "The version of the nonodo",
        }),
    };

    async run() {
        await this.config.runHook("check_folders", {})

        const { flags } = await this.parse(Run);

        const asyncController = new AbortController();

        try {
            const config = new Configuration()
            const configDir = this.config.configDir
            const isLoaded = await config.tryLoadFromDir(configDir)
            const logger = new Logger("Nonodo", Levels.INFO);
            const nonodoDir = this.config.dataDir;

            if (isLoaded && config.defaultVersion) {
                this.log("Configuration loaded");

                const cli = new CLI({
                    version: config.defaultVersion,
                });
                if (!cli.version) {
                    throw new Error("No default version found");
                }
                this.log(`Running brunodo ${cli.version} for ${arch()} ${platform()}`);
                const entry = config.versions.get(cli.version);
                if (!entry) {
                    throw new Error(`Version ${cli.version} not found`);
                }
                const binaryPath = join(nonodoDir, cli.binaryName);
                await runNonodo(binaryPath, asyncController, logger);

                return;
            }

            this.log("No default version found or configuration not loaded");
            const version = flags.version
            const cli = new CLI({
                version,
            });

            this.log(`Running brunodo ${cli.version} for ${arch()} ${platform()}`);

            const { path: nonodoPath, hash } = await getNonodoAvailable(
                asyncController.signal,
                nonodoDir,
                cli.url,
                cli.releaseName,
                cli.binaryName,
                logger,
            );
            config.addVersion(version, hash ?? "")
            await config.saveFile(configDir)
            await runNonodo(nonodoPath, asyncController, logger);
        } catch (e) {
            asyncController.abort(e);
            throw e;
        }

    }
}