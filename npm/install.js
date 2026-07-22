#!/usr/bin/env node
// Downloads the prebuilt bara binary for this platform from GitHub Releases
// and verifies it against the release checksums before installing.
"use strict";

const fs = require("fs");
const os = require("os");
const path = require("path");
const https = require("https");
const crypto = require("crypto");
const { execFileSync } = require("child_process");

const pkg = require("./package.json");
const REPO = "catmargiela/fuckhiggsfield";
const VERSION = pkg.version;

const PLATFORMS = { darwin: "darwin", linux: "linux", win32: "windows" };
const ARCHES = { x64: "amd64", arm64: "arm64" };

const platform = PLATFORMS[process.platform];
const arch = ARCHES[process.arch];

if (!platform || !arch) {
  console.error(
    `@catmargiela/bara: unsupported platform ${process.platform}/${process.arch}`
  );
  console.error("Supported: darwin|linux|windows x x64|arm64");
  process.exit(1);
}

const isWindows = platform === "windows";
const binName = isWindows ? "bara.exe" : "bara";
const ext = isWindows ? "zip" : "tar.gz";
const archive = `bara_${VERSION}_${platform}_${arch}.${ext}`;
const base = `https://github.com/${REPO}/releases/download/v${VERSION}`;

const vendorDir = path.join(__dirname, "vendor");

function get(url, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (redirects > 5) return reject(new Error("too many redirects"));
    https
      .get(url, { headers: { "User-Agent": "bara-npm-installer" } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          res.resume();
          return resolve(get(res.headers.location, redirects + 1));
        }
        if (res.statusCode !== 200) {
          res.resume();
          return reject(new Error(`HTTP ${res.statusCode} for ${url}`));
        }
        const chunks = [];
        res.on("data", (c) => chunks.push(c));
        res.on("end", () => resolve(Buffer.concat(chunks)));
        res.on("error", reject);
      })
      .on("error", reject);
  });
}

async function main() {
  fs.mkdirSync(vendorDir, { recursive: true });
  const archivePath = path.join(vendorDir, archive);

  console.log(`@catmargiela/bara: downloading ${base}/${archive}`);
  const body = await get(`${base}/${archive}`);
  fs.writeFileSync(archivePath, body);

  // Verify the download before unpacking anything.
  try {
    const sums = (await get(`${base}/checksums.txt`)).toString("utf8");
    const line = sums.split("\n").find((l) => l.trim().endsWith(archive));
    if (line) {
      const expected = line.trim().split(/\s+/)[0];
      const actual = crypto.createHash("sha256").update(body).digest("hex");
      if (expected !== actual) {
        throw new Error(
          `checksum mismatch for ${archive}\n  expected ${expected}\n  actual   ${actual}`
        );
      }
      console.log("@catmargiela/bara: checksum verified");
    } else {
      console.warn(`@catmargiela/bara: ${archive} not listed in checksums.txt`);
    }
  } catch (err) {
    // A missing checksums.txt is tolerable; a mismatch is not.
    if (String(err.message).includes("checksum mismatch")) {
      fs.unlinkSync(archivePath);
      throw err;
    }
    console.warn(`@catmargiela/bara: could not verify checksum — ${err.message}`);
  }

  if (isWindows) {
    // PowerShell ships with Windows; avoids adding an unzip dependency.
    execFileSync(
      "powershell",
      [
        "-NoProfile",
        "-NonInteractive",
        "-Command",
        `Expand-Archive -LiteralPath '${archivePath}' -DestinationPath '${vendorDir}' -Force`,
      ],
      { stdio: "inherit" }
    );
  } else {
    execFileSync("tar", ["-xzf", archivePath, "-C", vendorDir, binName]);
    fs.chmodSync(path.join(vendorDir, binName), 0o755);
  }

  fs.unlinkSync(archivePath);
  fs.writeFileSync(
    path.join(vendorDir, "install.json"),
    JSON.stringify({ version: VERSION, platform, arch, node: process.version, host: os.hostname() }, null, 2) + "\n"
  );
  console.log(`@catmargiela/bara: installed ${binName} ${VERSION}`);
}

main().catch((err) => {
  console.error("@catmargiela/bara: install failed —", err.message);
  process.exit(1);
});
