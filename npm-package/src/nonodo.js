#!/usr/bin/env node
"use strict";
import { existsSync, createReadStream, readFileSync, writeFileSync } from "node:fs";
import { Buffer } from "node:buffer";
import { URL } from "node:url";
import { spawn } from "node:child_process";
import { join } from "node:path";
import { arch, platform, tmpdir } from "node:os";
import { createHash } from "node:crypto";
import { get as request } from "node:https";
import { unzipSync } from "node:zlib";
import { SingleBar, Presets } from "cli-progress";
import AdmZip from "adm-zip";
import { Levels, Logger } from "./logger.js";
import { CLI } from "./cli.js";
import { getPlatform, getArch } from "./utils.js";

// const PACKAGE_NONODO_VERSION =
// process.env.PACKAGE_NONODO_VERSION ?? version;
// const PACKAGE_NONODO_URL = new URL(
//   process.env.PACKAGE_NONODO_URL ??
//     `https://github.com/Calindra/nonodo/releases/download/v${PACKAGE_NONODO_VERSION}/`,
// );
const PACKAGE_NONODO_DIR = process.env.PACKAGE_NONODO_DIR ?? tmpdir();

const HASH_ALGO = "md5";

const AVAILABLE_BINARY_NAME = new Set([
  "darwin-amd64",
  "darwin-arm64",
  "linux-amd64",
  "linux-arm64",
  "windows-amd64",
]);

const logger = new Logger("Nonodo", Levels.INFO);

/**
 *
 * @param {string} path
 * @param {string} algorithm
 * @returns {Promise<string>}
 */
function calculateHash(path, algorithm) {
  return new Promise((resolve, reject) => {
    const stream = createReadStream(path);
    const hash = createHash(algorithm);

    stream.on("data", (chunk) => {
      hash.update(chunk);
    });

    stream.on("error", (err) => {
      reject(err);
    });

    stream.on("end", () => {
      resolve(hash.digest("hex"));
    });
  });
}

function unpackZip(zipPath, destPath) {
  const zip = new AdmZip(zipPath);
  const entry = zip.getEntry("nonodo.exe");
  if (!entry) throw new Error("Dont find binary on zip");
  const buffer = entry.getData();
  writeFileSync(destPath, buffer, { mode: 0o755 });
}

function unpackTarball(tarballPath, destPath) {
  const tarballDownloadBuffer = readFileSync(tarballPath);
  const tarballBuffer = unzipSync(tarballDownloadBuffer);
  const data = extractFileFromTarball(tarballBuffer, "nonodo");
  if (!data) throw new Error("Dont find binary on tarball");
  writeFileSync(destPath, data, {
    mode: 0o755,
  });
}

/**
 *
 * @param {Buffer} tarballBuffer
 * @param {string} filepath
 * @returns
 */
function extractFileFromTarball(tarballBuffer, filepath) {
  // Tar archives are organized in 512 byte blocks.
  // Blocks can either be header blocks or data blocks.
  // Header blocks contain file names of the archive in the first 100 bytes, terminated by a null byte.
  // The size of a file is contained in bytes 124-135 of a header block and in octal format.
  // The following blocks will be data blocks containing the file.

  let offset = 0;
  while (offset < tarballBuffer.length) {
    const header = tarballBuffer.slice(offset, offset + 512);
    offset += 512;

    const fileName = header.toString("utf-8", 0, 100).replace(/\0.*/g, "");
    const fileSize = parseInt(
      header.toString("utf-8", 124, 136).replace(/\0.*/g, ""),
      8,
    );

    if (fileName === filepath) {
      return tarballBuffer.subarray(offset, offset + fileSize);
    }

    // Clamp offset to the uppoer multiple of 512
    offset = (offset + fileSize + 511) & ~511;
  }
}

async function downloadBinary(signal, nonodoUrl, releaseName) {
  if (!(nonodoUrl instanceof URL)) {
    throw new Error("Invalid URL");
  }
  const dir = PACKAGE_NONODO_DIR;
  const url = new URL(nonodoUrl);
  if (!url.href.endsWith("/")) url.pathname += "/";
  url.pathname += releaseName;

  logger.info(`Downloading: ${url.href}`);

  const dest = join(dir, releaseName);

  const binary = await makeRequest(signal, url);

  writeFileSync(dest, binary, {
    signal,
  });
}

async function downloadHash(signal, nonodoUrl, releaseName) {
  if (!(nonodoUrl instanceof URL)) {
    throw new Error("Invalid URL");
  }

  const algo = HASH_ALGO;
  const filename = `${releaseName}.${algo}`;

  const dir = PACKAGE_NONODO_DIR;
  const url = new URL(nonodoUrl);
  if (!url.href.endsWith("/")) url.pathname += "/";
  url.pathname += filename;

  logger.info(`Downloading: ${url.href}`);

  const dest = join(dir, filename);

  const response = await makeRequest(signal, url);
  const body = response.toString("utf-8");

  writeFileSync(dest, body, {
    signal,
  });

  logger.info(`Downloaded hex: ${dest}`);

  return body.trim();
}

/**
 *
 * @param {URL} url
 * @returns {Promise<Buffer>}
 */
function makeRequest(signal, url) {
  return new Promise((resolve, reject) => {
    /** @type {SingleBar=} */
    let bar;

    const req = request(url, (res) => {
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
        makeRequest(signal, new URL(res.headers.location)).then(resolve).catch(reject);
        // Error
      } else {
        bar?.stop();
        reject(
          new Error(`Error ${res.statusCode} when downloading the package: ${res.statusMessage}`),
        );
      }
    });
    req.on("error", (e) => {
      bar?.stop();
      reject(e);
    });

    signal.addEventListener("abort", () => {
      req.destroy();
      reject(new Error("Request aborted."));
    });
  });
}

async function runNonodo(location) {
  logger.info(`Running brunodo binary: ${location}`);

  const args = process.argv.slice(2);
  const nonodoBin = spawn(location, args, { stdio: "inherit" });
  nonodoBin.on("exit", (code, signal) => {
    process.on("exit", () => {
      if (signal) {
        process.kill(process.pid, signal);
      } else {
        process.exit(code ?? 1);
      }
    });
  });

  process.on("SIGINT", function () {
    nonodoBin.kill("SIGINT");
    nonodoBin.kill("SIGTERM");
  });
}

async function getNonodoAvailable(signal, nonodoUrl, releaseName, binaryName) {
  const nonodoPath = PACKAGE_NONODO_DIR;

  const myPlatform = getPlatform();
  const myArch = getArch();
  const support = `${myPlatform}-${myArch}`;

  if (AVAILABLE_BINARY_NAME.has(support)) {
    logger.info(`Platform supported: ${support}`);
    const binaryPath = join(nonodoPath, binaryName);

    if (existsSync(binaryPath)) return binaryPath;

    logger.info(`Nonodo binary not found: ${binaryPath}`);
    logger.info(`Downloading nonodo binary...`);
    const [hash] = await Promise.all([downloadHash(signal, nonodoUrl, releaseName), downloadBinary(signal, nonodoUrl, releaseName)]);

    logger.info(`Downloaded nonodo binary.`);
    logger.info(`Verifying hash...`);

    const releasePath = join(nonodoPath, releaseName);
    const calculatedHash = await calculateHash(releasePath, HASH_ALGO);

    if (hash !== calculatedHash) {
      throw new Error(
        `Hash mismatch for nonodo binary. Expected ${hash}, got ${calculatedHash}`,
      );
    }

    logger.info(`Hash verified.`);

    if (getPlatform() !== "windows") {
      unpackTarball(releasePath, binaryPath);
    } else {
      /** unzip this */
      unpackZip(releasePath, binaryPath);
    }

    if (!existsSync(binaryPath)) throw new Error("Problem on unpack");

    return binaryPath;
  }

  throw new Error(`Incompatible platform.`);
}


async function tryPackageNonodo() {
  const asyncController = new AbortController();

  try {
    const cli = new CLI({
      version: "2.1.1-beta"
    });

    logger.info(`Running brunodo ${cli.version} for ${arch()} ${platform()}`);

    process.once("SIGINT", () => asyncController.abort());
    const nonodoPath = await getNonodoAvailable(asyncController.signal, cli.url, cli.releaseName, cli.binaryName);
    logger.info(`nonodo path: ${nonodoPath}`);
    await runNonodo(nonodoPath);
    return true;
  } catch (e) {
    asyncController.abort(e);
    throw e;
  }
}

tryPackageNonodo()
  .then((success) => {
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
