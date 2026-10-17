"use strict";
const { chromium } = require("playwright");
const { spawn } = require("node:child_process");
const { mkdtemp, readFile, rm, mkdir } = require("node:fs/promises");
const { tmpdir } = require("node:os");
const { resolve, join } = require("node:path");
const assert = require("node:assert/strict");

(async () => {
  const directory = await mkdtemp(join(tmpdir(), "akari-development-"));
  const artifacts = join(__dirname, "artifacts");
  await mkdir(artifacts, { recursive: true });
  let host, exited, address, token, browser;
  async function start() {
    host = spawn(resolve(process.argv[2]), [], {
      env: { ...process.env, AKARI_DATA_DIR: directory, AKARI_ADDR: "127.0.0.1:0", AKARI_ADMIN_ADDR: "127.0.0.1:0", AKARI_ADMIN_AUTH_DISABLED: "false", AKARI_DOCS_REPO: "" },
      stdio: ["ignore", "pipe", "pipe"],
    });
    exited = new Promise((done) => host.once("exit", (code) => done(code)));
    let logs = "";
    await new Promise((done, fail) => {
      const timeout = setTimeout(() => fail(new Error("Development host startup timed out")), 10000);
      const capture = (chunk) => {
        logs += chunk.toString();
        if (logs.includes("foundation host listening")) { clearTimeout(timeout); done(); }
      };
      host.stderr.on("data", capture);
      host.stdout.on("data", capture);
      host.once("error", (err) => { clearTimeout(timeout); fail(err); });
      host.once("exit", () => { clearTimeout(timeout); fail(new Error("Development host exited before startup")); });
    });
    address = `http://${logs.match(/management console listening.*address=(127\.0\.0\.1:\d+)/)[1]}`;
    token = (await readFile(join(directory, "admin.token"), "utf8")).trim();
  }
  async function stop(signal = "SIGTERM") {
    if (!host) return;
    host.kill(signal);
    const deadline = setTimeout(() => host.kill("SIGKILL"), 6000);
    const code = await exited;
    clearTimeout(deadline);
    host = null;
    if (signal === "SIGTERM") assert.equal(code, 0, "Host did not stop cleanly");
  }
  async function login(page, hash = "tasks") {
    await page.goto(`${address}/#${hash}`);
    await page.locator("#code").fill(token);
    await page.locator("#login-form button").click();
    await page.locator("#app:not([hidden])").waitFor();
  }
  async function mcp(name, args) {
    const response = await fetch(`${address}/mcp`, {
      method: "POST", headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json", Accept: "application/json, text/event-stream", "MCP-Protocol-Version": "2025-11-25" },
      body: JSON.stringify({ jsonrpc: "2.0", id: 1, method: "tools/call", params: { name, arguments: args } }),
    });
    assert.equal(response.status, 200);
    const result = await response.json();
    assert.equal(result.error, undefined, `MCP ${name} protocol error`);
    assert.equal(result.result.isError, undefined, `MCP ${name} tool error`);
    return result.result.structuredContent;
  }
  try {
    await start();
    browser = await chromium.launch({ executablePath: process.env.AKARI_CHROMIUM || undefined, headless: true, args: ["--no-sandbox"] });
    const context = await browser.newContext({ viewport: { width: 1500, height: 1050 }, reducedMotion: "reduce" });
    const page = await context.newPage();
    const errors = [], external = [];
    page.on("pageerror", (err) => errors.push(err.message));
    page.on("request", (request) => { if (!request.url().startsWith(`${address}/`) && !request.url().startsWith("data:")) external.push(request.url()); });
    await login(page);
    await page.locator(".task-card").first().waitFor();
    assert.ok(await page.locator(".task-card").count() >= 20);
    await page.locator("#task-filter-status").selectOption("decision");
    assert.ok(await page.locator(".task-column-decision .task-card").count() >= 5);
    await page.locator("#task-search").fill("not-found-task");
    await page.locator("#task-empty:not([hidden])").waitFor();
    await page.locator("#task-search").fill("");
    await page.locator("#task-filter-status").selectOption("active");
    await page.screenshot({ path: join(artifacts, "development-tasks.png"), fullPage: true });

    await page.locator("#task-new").click();
    const title = '<img src="https://invalid.example/task"> API task';
    await page.locator("#task-title").fill(title);
    await page.locator("#task-description").fill("UI and MCP use the same database.");
    await page.locator("#task-notes").fill("created in console");
    await page.locator("#task-priority").selectOption("high");
    await page.locator("#task-references").selectOption("design/05-delivery.md");
    await page.locator("#task-save").click();
    await page.locator("#task-dialog").waitFor({ state: "hidden" });
    const board = await mcp("tasks_list", {});
    const task = board.tasks.find((value) => value.title === title);
    assert.ok(task);
    assert.equal(task.notes, "created in console");
    assert.equal(await page.locator('img[src="https://invalid.example/task"]').count(), 0);

    await page.locator(`.task-card[data-task-id="${task.id}"] .task-title`).click();
    await page.locator("#task-notes").fill("unsaved conflict draft");
    const edited = { ...task, status: "in_progress", notes: "updated by MCP" };
    await mcp("tasks_update", { task: edited });
    await page.locator("#task-save").click();
    await page.locator("#task-form-error:not([hidden])").waitFor();
    assert.equal(await page.locator("#task-notes").inputValue(), "unsaved conflict draft");
    assert.ok((await page.locator("#task-form-error").textContent()).includes("更新"));
    await page.locator("#task-cancel").click();
    await page.locator("#tasks-refresh").click();
    await page.waitForFunction((id) => document.querySelector(`.task-card[data-task-id="${id}"] select`)?.value === "in_progress", task.id);
    assert.equal((await mcp("tasks_get", { id: task.id })).notes, "updated by MCP");

    const download = page.waitForEvent("download");
    await page.locator("#tasks-export").click();
    const exported = JSON.parse(await readFile(await (await download).path(), "utf8"));
    assert.ok(exported.tasks.some((value) => value.id === task.id));
    await page.setViewportSize({ width: 390, height: 844 });
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), "Mobile task board overflow");
    await page.screenshot({ path: join(artifacts, "development-mobile.png"), fullPage: true });
    await page.setViewportSize({ width: 1500, height: 1050 });

    await page.locator("[data-nav=docs]").click();
    await page.locator("#doc-new").click();
    await page.locator("#doc-edit-path").fill("design/browser-note.md");
    await page.locator("#doc-content").fill("# Browser note\n\nCommitted from the console.\n");
    await page.locator("#doc-save").click();
    await page.locator("#doc-dialog").waitFor({ state: "hidden" });
    await page.locator("#article h1").filter({ hasText: "Browser note" }).waitFor();
    const document = await mcp("documents_read", { path: "design/browser-note.md" });
    assert.ok(document.content.includes("Committed from the console."));
    assert.equal(document.revision.length, 40);
    const previous = document.revision;
    await page.locator("#doc-edit").click();
    await page.locator("#doc-content").fill("# Browser note\n\nEdited twice.\n");
    await page.locator("#doc-save").click();
    await page.locator("#doc-dialog").waitFor({ state: "hidden" });
    await page.locator("#article").filter({ hasText: "Edited twice." }).waitFor();
    assert.notEqual((await mcp("documents_read", { path: document.path })).revision, previous);
    await page.screenshot({ path: join(artifacts, "git-document.png"), fullPage: true });

    await stop("SIGKILL");
    await start();
    assert.equal((await mcp("tasks_get", { id: task.id })).notes, "updated by MCP");
    assert.ok((await mcp("documents_read", { path: document.path })).content.includes("Edited twice."));
    await login(page);
    await page.locator(`.task-card[data-task-id="${task.id}"] .task-title`).click();
    page.once("dialog", (dialog) => dialog.accept());
    await page.locator("#task-delete").click();
    await page.locator("#task-dialog").waitFor({ state: "hidden" });
    assert.equal((await mcp("tasks_list", {})).tasks.some((value) => value.id === task.id), false);
    await page.locator("#logout").click();
    await page.locator("#login:not([hidden])").waitFor();
    assert.equal((await context.request.get(`${address}/api/development/tasks`)).status(), 401);
    assert.deepEqual(errors, [], "Development UI JavaScript errors");
    assert.deepEqual(external, [], "External requests from development UI");
    console.log("Development browser passed: task CRUD, MCP/API/console sharing, stale-edit draft retention, JSON export, mobile, Git commits, SIGKILL persistence, logout, no external requests");
  } finally {
    if (browser) await browser.close();
    await stop();
    await rm(directory, { recursive: true, force: true });
  }
})().catch((error) => { console.error(error.stack); process.exitCode = 1; });
