"use strict";
import { SingleBar, Presets } from "cli-progress";
import { Buffer } from "node:buffer";
import { get as request } from "node:https";
import { arch, platform } from "node:os";
import { URL } from "node:url";

export function getPlatform() {
  const plat = platform();
  if (plat === "win32") return "windows";
  else return plat;
}
export function getArch() {
  const arc = arch();
  if (arc === "x64") return "amd64";
  else return arc;
}

/**
 * @param {AbortSignal} signal
 * @param {URL} url
 * @returns {Promise<Buffer>}
 */
export function makeRequest(signal, url) {
  return new Promise((resolve, reject) => {
    /** @type {SingleBar=} */
    let bar;

    /** @typedef {import("node:https").RequestOptions} */
    const options = {
      hostname: url.hostname,
      port: url.port,
      path: url.pathname + url.search,
      method: "GET",
      headers: {
        "User-Agent": "node",
      },
      signal,
    };

    const req = request(options, (res) => {
      if (!res.statusCode) {
        reject(new Error("No status code"));
        return;
      }

      // Ok
      if (res.statusCode >= 200 && res.statusCode < 300) {
        const contentLength = res.headers["content-length"];
        const chunks = [];
        let size = 0;
        if (contentLength) {
          const length = parseInt(contentLength, 10);
          if (!Number.isNaN(length)) {
            bar = new SingleBar({}, Presets.shades_classic);
            bar.start(length, 0);
          }
        }

        res.on("data", (chunk) => {
          chunks.push(chunk);
          size += chunk.length;
          bar?.update(size);
        });

        res.on("end", () => {
          bar?.stop();
          resolve(Buffer.concat(chunks));
        });
        // Redirect
      } else if (
        res.statusCode >= 300 &&
        res.statusCode < 400 &&
        res.headers.location
      ) {
        makeRequest(signal, new URL(res.headers.location))
          .then(resolve)
          .catch(reject);
        // Error
      } else {
        bar?.stop();
        reject(
          new Error(
            `Error ${res.statusCode} when downloading the package: ${res.statusMessage}`,
          ),
        );
      }
    });
    req.on("error", (e) => {
      bar?.stop();
      reject(e);
    });

    // signal.addEventListener("abort", () => {
    //   req.destroy();
    //   reject(new Error("Request aborted."));
    // });
  });
}

export async function listTags(signal, logger) {
  const repo = "nonodo";
  const namespace = "calindra";
  const url = new URL(`https://api.github.com/repos/${namespace}/${repo}/tags`);
  logger.info(`Requesting tags from ${url}`);
  const res = await makeRequest(signal, url);
  const tags = JSON.parse(res.toString());
  logger.debug(tags);
  if (!Array.isArray(tags)) {
    throw new Error("Invalid response");
  }
  const names = tags.map((tag) => tag.name);
  logger.info(`Valid tags: ${names.join(", ")}`);
  return names;
}
