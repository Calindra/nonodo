import { Command, handle, Flags, flush } from "@oclif/core";
import { listTags } from "../utils.js";
import { Levels, Logger } from "../logger.js";
import generateTable from "tty-table"

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

      const tags = await listTags(abortCtrl.signal, logger);
      const headers = [{ value: "Version" }, { value: "SHA Commit" }]
      const rows = tags.map((tag) => [tag.name, tag.commit.sha])
      const table = generateTable(headers, rows, { compact: true })

      this.log(table.render())
    } catch (error) {
      abortCtrl.abort(error);
      throw error;
    }
  }
}
