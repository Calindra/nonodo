import { Command, Flags, Args } from "@oclif/core";
import { valid, parse } from "semver"
import { existsSync } from "node:fs"

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
        const version = parse(args.version);
        if (version === null) {
            throw new Error(`Invalid version: ${args.version}`);
        }
        console.log("Hello from install");
    }
}