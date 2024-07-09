import { Command, Flags } from "@oclif/core";
import { listTags } from "../utils.js";
import { Levels, Logger } from "../logger.js";
import generateTable from "tty-table"
import { Configuration } from "../config.js";
import { parse } from "semver";

export class ListTags extends Command {
  static description = "List all tags";
  static flags = {
    help: Flags.help({ char: "h" }),
    debug: Flags.boolean({ char: "d", description: "Show debug information" }),
  };

  async run() {
    const abortCtrl = new AbortController();

    try {
      const { flags } = await this.parse(ListTags);
      const level = flags.debug ? Levels.DEBUG : Levels.INFO;
      const logger = new Logger("Tags", level);

      const configDir = this.config.configDir
      const config = new Configuration()
      await config.tryLoadFromDir(configDir)

      const tags = await listTags(abortCtrl.signal, logger);
      const headers = [{ value: "version" }, { value: "sha_commit" }, { value: "installed" }]
      const rows = tags.map((tag) => {
        const version = parse(tag.name)?.version ?? ""
        return ({ version, sha_commit: tag.commit.sha, installed: config.versions.has(version) });
      })
      const table = generateTable(headers, rows, { compact: true })

      this.log(table.render())
    } catch (error) {
      abortCtrl.abort(error);
      throw error;
    }
  }
}
