import { Command, handle, Flags, flush } from "@oclif/core";
import { listTags } from "../utils.js";
import { Levels, Logger } from "../logger.js";

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
      const logger = new Logger("ListTags", level);
      await listTags(abortCtrl.signal, logger);
      await flush();
    } catch (error) {
      abortCtrl.abort(error);
      await handle(error);
    }
  }
}
