"use strict";
const { chromium } = require("playwright");
const { spawn } = require("node:child_process");
const { mkdtemp, readFile, rm, mkdir } = require("node:fs/promises");
const { tmpdir } = require("node:os");
const { resolve, join } = require("node:path");
const assert = require("node:assert/strict");

(async () => {
  const directory = await mkdtemp(join(tmpdir(), "akari-console-"));
  const artifacts = resolve(__dirname, "artifacts");
  await mkdir(artifacts, { recursive: true });
  const host = spawn(resolve(process.argv[2]), [], {
    env: {
      ...process.env,
      AKARI_ADDR: "127.0.0.1:0",
      AKARI_ADMIN_ADDR: "127.0.0.1:0",
      AKARI_DATA_DIR: directory,
    },
    stdio: ["ignore", "pipe", "pipe"],
  });
  let logs = "",
    browser;
  const exited = new Promise((done) => host.once("exit", (code) => done(code)));
  try {
    await new Promise((done, fail) => {
      const timeout = setTimeout(
        () => fail(new Error("Host startup timed out")),
        10000,
      );
      host.once("error", fail);
      const capture = (chunk) => {
        logs += chunk.toString();
        if (logs.includes("foundation host listening")) {
          clearTimeout(timeout);
          done();
        }
      };
      host.stderr.on("data", capture);
      host.stdout.on("data", capture);
      host.once("exit", (code) => {
        clearTimeout(timeout);
        if (code !== null)
          fail(new Error(`Host exited during startup: ${code}`));
      });
    });
    const address = logs.match(
      /management console listening.*address=(127\.0\.0\.1:\d+)/,
    )[1];
    const publicAddress = logs.match(
      /foundation host listening.*address=(127\.0\.0\.1:\d+)/,
    )[1];
    const token = (
      await readFile(join(directory, "admin.token"), "utf8")
    ).trim();
    browser = await chromium.launch({
      executablePath: process.env.AKARI_CHROMIUM || undefined,
      headless: true,
      args: ["--no-sandbox"],
    });
    const context = await browser.newContext({
      viewport: { width: 1440, height: 1050 },
      reducedMotion: "reduce",
    });
    const page = await context.newPage();
    const errors = [],
      externalRequests = [];
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("request", (request) => {
      if (
        !request.url().startsWith(`http://${address}/`) &&
        !request.url().startsWith("data:")
      )
        externalRequests.push(request.url());
    });
    assert.equal(
      (await context.request.get(`http://${address}/api/status`)).status(),
      401,
    );
    assert.equal(
      (
        await context.request.get(`http://${publicAddress}/api/status`)
      ).status(),
      404,
    );
    await page.goto(`http://${address}/`);
    await page.locator("#code").fill("incorrect-code");
    await page.locator("#login-form button").click();
    await page
      .locator("#login-error")
      .filter({ hasText: "管理用コード" })
      .waitFor();
    await page.locator("#code").fill(token);
    await page.locator("#login-form button").click();
    await page.locator("#app:not([hidden])").waitFor();
    assert.equal(await page.locator(".graph-node").count(), 13);
    await page.mouse.move(0, 0);
    await page.screenshot({
      path: join(artifacts, "design-map.png"),
      fullPage: true,
    });
    await page
      .locator('.graph-node[data-path="docs/01-vision.md"]')
      .press("Enter");
    await page.locator("#article h1").filter({ hasText: "ビジョン" }).waitFor();
    await page.locator("[data-nav=map]").click();
    await page.locator("[data-group=design]").click();
    assert.ok((await page.locator(".graph-node").count()) >= 10);
    await page.locator("#zoom-in").click();
    assert.equal(await page.locator("#zoom-reset").textContent(), "125%");
    await page.locator("#zoom-reset").click();
    await page.locator("[data-nav=docs]").click();
    await page.locator("#search").fill("zzz-no-document-match");
    await page.locator("#no-results:not([hidden])").waitFor();
    await page.locator("#search").fill("思考アーキテクチャ");
    assert.ok((await page.locator(".doc-card").count()) > 0);
    const docs = await page.evaluate(async () =>
      (await fetch("/api/docs")).json(),
    );
    const diagramDocs = docs.filter((doc) =>
      doc.content.includes("```mermaid"),
    );
    for (const doc of diagramDocs) {
      await page.evaluate((path) => {
        location.hash = `doc=${encodeURIComponent(path)}`;
      }, doc.path);
      const count = (doc.content.match(/```mermaid/g) || []).length;
      await page.waitForFunction(
        (expected) =>
          document.querySelectorAll(".diagram-body svg").length === expected,
        count,
        { timeout: 30000 },
      );
      assert.equal(
        await page
          .locator(".diagram-body")
          .filter({ hasText: "描画できません" })
          .count(),
        0,
        doc.path,
      );
      assert.ok((await page.locator("#toc a").count()) > 0);
      if (doc.path === "docs/06-architecture.md") {
        await page.screenshot({
          path: join(artifacts, "architecture-reader.png"),
          fullPage: true,
        });
        const width = await page
          .locator(".diagram-body svg")
          .evaluate((svg) => svg.style.width);
        await page
          .getByRole("button", { name: "図を拡大", exact: true })
          .click();
        assert.notEqual(
          await page
            .locator(".diagram-body svg")
            .evaluate((svg) => svg.style.width),
          width,
        );
      }
    }
    await page.locator("#toc a").first().click();
    await page.waitForFunction(() => location.hash.includes("section="));
    await page.locator("#related a").first().click();
    await page.locator("#article h1").waitFor();
    await page.locator("[data-nav=status]").click();
    await page.locator(".runtime-stat strong").first().waitFor();
    assert.equal(
      await page.locator(".runtime-stat strong").first().textContent(),
      "0",
    );
    await page.locator("#refresh").click();
    await page.screenshot({
      path: join(artifacts, "runtime-status.png"),
      fullPage: true,
    });
    await page.setViewportSize({ width: 390, height: 844 });
    await page.locator("[data-nav=docs]").click();
    await page.screenshot({
      path: join(artifacts, "mobile-library.png"),
      fullPage: true,
    });
    assert.ok(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth + 1,
      ),
      "Mobile layout overflow",
    );
    await page.locator("#logout").click();
    await page.locator("#login:not([hidden])").waitFor();
    assert.equal(
      (await context.request.get(`http://${address}/api/status`)).status(),
      401,
    );
    assert.deepEqual(errors, [], "Browser JavaScript errors");
    assert.deepEqual(externalRequests, [], "External browser requests");
    console.log(
      `Console browser passed: authentication, map, search, ${diagramDocs.length} Mermaid documents, links, state, mobile, logout, no external requests`,
    );
  } finally {
    if (browser) await browser.close();
    host.kill("SIGTERM");
    const deadline = setTimeout(() => host.kill("SIGKILL"), 6000);
    const code = await exited;
    clearTimeout(deadline);
    await rm(directory, { recursive: true, force: true });
    if (code !== 0) process.exitCode = 1;
  }
})().catch((error) => {
  console.error(error.message);
  process.exitCode = 1;
});
