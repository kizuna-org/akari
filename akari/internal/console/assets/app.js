"use strict";
(() => {
  const $ = (id) => document.getElementById(id);
  const state = {
    documents: [],
    documentRevision: "",
    links: [],
    group: "docs",
    filter: "all",
    zoom: 100,
    render: 0,
  };
  const text = (tag, value, className = "") => {
    const el = document.createElement(tag);
    el.textContent = value;
    el.className = className;
    return el;
  };
  const go = (path, section = "") =>
    `#doc=${encodeURIComponent(path)}${section ? `&section=${encodeURIComponent(section)}` : ""}`;
  const lookup = (path) => state.documents.find((doc) => doc.path === path);
  let documentRevision = "";
  let documentSaving = false;
  const interruptedDialogs = new Set();
  const shortTitle = (doc) =>
    doc.path.endsWith("/README.md")
      ? "読み方とドキュメント目次"
      : doc.title
          .replace(/^\d+[.．]?\s*/, "")
          .replace(/（.*?）/g, "")
          .replace(/^Akari\s*[—ー]?\s*/, "");
  const resolve = (path, target) => {
    try {
      const url = new URL(target, `https://documentation.local/${path}`);
      if (url.origin !== "https://documentation.local") return null;
      return {
        path: decodeURIComponent(url.pathname.slice(1)),
        section: decodeURIComponent(url.hash.slice(1)),
      };
    } catch {
      return null;
    }
  };
  async function api(path, options = {}) {
    const response = await fetch(path, {
      credentials: "same-origin",
      ...options,
    });
    if (response.status === 401) {
      showLogin();
      throw new Error("管理用コードを確認してログインしてください。");
    }
    if (response.ok && path === "/api/docs" && !options.method)
      state.documentRevision = (response.headers.get("ETag") || "").replaceAll('"', "");
    if (!response.ok && path.startsWith("/api/development/tasks")) {
      const messages = {
        409: "別の画面で更新されています。下書きは保持しています。一覧を更新して内容を確認し、編集を開き直してください。",
        400: "入力を確認してください。タイトル・状態・分類と、資料参照は8件以内で指定してください。",
        404: "タスクが見つかりません。一覧を更新してください。",
        403: "同じコンソールから操作してください。",
      };
      throw new Error(messages[response.status] || "保存状態を確認できません。一覧を更新して確認してください。");
    }
    if (!response.ok && path === "/api/docs" && options.method === "PUT") {
      if (response.status === 409) throw new Error("資料が更新されたか、Gitに未コミットの変更があります。下書きを控え、資料を更新してから編集を開き直してください。");
      if (response.status === 400) throw new Error("ファイル名はdocs/またはdesign/直下の英数字・ハイフン・下線と.mdで指定し、本文は1MB以内にしてください。");
      throw new Error("Gitへの保存を確認できません。資料を更新して、コミットと本文を確認してください。");
    }
    if (!response.ok)
      throw new Error(
        "情報を取得できませんでした。ホストの接続を確認してください。",
      );
    return response.status === 204 ? null : response.json();
  }
  function showLogin(preserveDraft = true) {
    if (!preserveDraft) interruptedDialogs.clear();
    document.querySelectorAll("dialog[open]").forEach((dialog) => {
      if (preserveDraft) interruptedDialogs.add(dialog);
      dialog.close();
    });
    $("login").hidden = false;
    $("app").hidden = true;
  }
  function error(message) {
    $("app-error").textContent = message;
    $("app-error").hidden = false;
    $("connection").textContent = "接続を確認";
  }
  async function initialize() {
    state.documents = (await api("/api/docs")).map((doc) => ({
      ...doc,
      title: doc.content.match(/^# (.+)$/m)?.[1] || doc.path,
      group: doc.path.split("/")[0],
      number: doc.path.match(/\/(\d+)/)?.[1] || "↗",
    }));
    const unique = new Set();
    state.links = [];
    for (const doc of state.documents) {
      for (const match of doc.content.matchAll(/\]\(([^)]+)\)/g)) {
        const target = resolve(doc.path, match[1]);
        if (!target || target.path === doc.path || !lookup(target.path))
          continue;
        const key = `${doc.path}>${target.path}`;
        if (!unique.has(key)) {
          state.links.push({ from: doc.path, to: target.path });
          unique.add(key);
        }
      }
    }
    $("doc-count").textContent = state.documents.length;
    $("link-count").textContent = state.links.length;
    $("login").hidden = true;
    $("app").hidden = false;
    if (document.body.dataset.authDisabled === "true") $("logout").hidden = true;
    $("code").value = "";
    $("app-error").hidden = true;
    mermaid.initialize({
      startOnLoad: false,
      securityLevel: "strict",
      theme: "base",
      suppressErrorRendering: true,
      flowchart: { htmlLabels: false, useMaxWidth: false },
      themeVariables: {
        primaryColor: "#edf1e6",
        primaryTextColor: "#374637",
        primaryBorderColor: "#a9b69d",
        lineColor: "#8c9e80",
        secondaryColor: "#f6ede3",
        tertiaryColor: "#f8f9f4",
        fontFamily: "sans-serif",
      },
    });
    renderMap();
    renderLibrary();
    await route();
    for (const dialog of interruptedDialogs) dialog.showModal();
    interruptedDialogs.clear();
  }
  function svg(tag, attrs = {}, value = "") {
    const node = document.createElementNS("http://www.w3.org/2000/svg", tag);
    for (const [key, val] of Object.entries(attrs))
      node.setAttribute(key, String(val));
    if (value) node.textContent = value;
    return node;
  }
  function renderMap() {
    document
      .querySelectorAll(".map-explanation .legend-dot")
      .forEach((dot, index) => {
        dot.hidden = state.group === "design" && index !== 2;
        dot.nextElementSibling.hidden = dot.hidden;
        if (index === 2)
          dot.nextElementSibling.textContent =
            state.group === "design" ? "技術設計" : "原則・能力・発展";
      });
    const docs = state.documents.filter((doc) => doc.group === state.group);
    const positions = new Map();
    let index = 0;
    for (const doc of docs) {
      const overview = doc.path.endsWith("/README.md");
      positions.set(
        doc.path,
        overview
          ? { x: 430, y: 28 }
          : { x: 34 + (index % 4) * 260, y: 160 + Math.floor(index / 4) * 145 },
      );
      if (!overview) index++;
    }
    const graph = $("graph");
    graph.replaceChildren();
    const links = state.links.filter(
      (link) => positions.has(link.from) && positions.has(link.to),
    );
    const layer = svg("g");
    graph.append(layer);
    for (const link of links) {
      const from = positions.get(link.from),
        to = positions.get(link.to);
      const x1 = from.x + 110,
        y1 = from.y + 40,
        x2 = to.x + 110,
        y2 = to.y + 40;
      const path = svg("path", {
        d: `M ${x1} ${y1} Q ${(x1 + x2) / 2} ${(y1 + y2) / 2 - 40} ${x2} ${y2}`,
        class: "graph-edge",
        "data-from": link.from,
        "data-to": link.to,
      });
      path.append(
        svg(
          "title",
          {},
          `${shortTitle(lookup(link.from))} → ${shortTitle(lookup(link.to))}`,
        ),
      );
      path.addEventListener("mouseenter", () => highlight(link.from, link.to));
      path.addEventListener("mouseleave", () => highlight());
      layer.append(path);
    }
    for (const doc of docs) {
      const { x, y } = positions.get(doc.path);
      const node = svg("g", {
        transform: `translate(${x},${y})`,
        class: "graph-node",
        tabindex: 0,
        role: "button",
        "aria-label": doc.title,
        "data-path": doc.path,
      });
      node.append(svg("rect", { width: 220, height: 80, rx: 7 }));
      const inner =
        doc.group === "docs" && ["02", "03", "04", "05"].includes(doc.number);
      const mechanism =
        doc.group === "docs" && ["06", "07", "08"].includes(doc.number);
      node.append(
        svg("circle", {
          cx: 18,
          cy: 18,
          r: 3,
          fill: inner ? "#bb785d" : mechanism ? "#729b8b" : "#969e9b",
        }),
      );
      node.append(
        svg("text", { x: 30, y: 21, class: "node-number" }, doc.number),
      );
      const title = shortTitle(doc);
      node.append(
        svg(
          "text",
          { x: 17, y: 43, class: "node-title" },
          title.length > 17 ? title.slice(0, 16) + "…" : title,
        ),
      );
      node.append(svg("text", { x: 17, y: 64, class: "node-sub" }, doc.path));
      node.append(svg("text", { x: 196, y: 23, class: "node-sub" }, "↗"));
      node.addEventListener("click", () => {
        location.hash = go(doc.path);
      });
      node.addEventListener("keydown", (event) => {
        if (["Enter", " "].includes(event.key)) {
          event.preventDefault();
          location.hash = go(doc.path);
        }
      });
      node.addEventListener("mouseenter", () => highlight(doc.path));
      node.addEventListener("focus", () => highlight(doc.path));
      node.addEventListener("mouseleave", () => highlight());
      node.addEventListener("blur", () => highlight());
      graph.append(node);
    }
    $("map-caption").textContent = `${state.group} / ${docs.length} documents`;
    function highlight(path, to) {
      const neighbors = new Set([path, to]);
      graph.querySelectorAll(".graph-edge").forEach((edge) => {
        const active =
          path &&
          (to
            ? edge.dataset.from === path && edge.dataset.to === to
            : [edge.dataset.from, edge.dataset.to].includes(path));
        edge.classList.toggle("focused", Boolean(active));
        edge.style.opacity = path ? (active ? "1" : ".12") : ".55";
        if (active) {
          neighbors.add(edge.dataset.from);
          neighbors.add(edge.dataset.to);
        }
      });
      graph
        .querySelectorAll(".graph-node")
        .forEach((node) =>
          node.classList.toggle(
            "dim",
            Boolean(path && !neighbors.has(node.dataset.path)),
          ),
        );
      $("graph-detail").textContent = path
        ? to
          ? `${shortTitle(lookup(path))} → ${shortTitle(lookup(to))}`
          : `${shortTitle(lookup(path))} と関連するドキュメント`
        : "リンクはドキュメント本文にある参照です。";
    }
    highlight();
  }
  function renderLibrary() {
    const query = $("search").value.trim().toLocaleLowerCase();
    const docs = state.documents.filter(
      (doc) =>
        (state.filter === "all" || doc.group === state.filter) &&
        (!query ||
          `${doc.title}\n${doc.content}`.toLocaleLowerCase().includes(query)),
    );
    $("doc-list").replaceChildren();
    $("search-count").textContent = `${docs.length} 件`;
    $("no-results").hidden = docs.length > 0;
    for (const doc of docs) {
      const link = document.createElement("a");
      link.className = "doc-card";
      link.href = go(doc.path);
      link.append(text("span", doc.number, "chapter"));
      const body = document.createElement("div");
      body.append(text("h2", shortTitle(doc)));
      body.append(text("p", doc.path));
      const diagrams = (doc.content.match(/\x60\x60\x60mermaid/g) || []).length;
      body.append(
        text(
          "span",
          `${doc.group === "docs" ? "振る舞いの仕様" : "技術設計"}${diagrams ? ` · 図 ${diagrams}点` : ""}`,
          "tag",
        ),
      );
      link.append(body);
      $("doc-list").append(link);
    }
  }
  async function route() {
    if ($("app").hidden) return;
    $("app-error").hidden = true;
    const hash = location.hash.slice(1) || "map";
    const isDoc = hash.startsWith("doc=");
    const view = isDoc
      ? "reader"
      : ["map", "docs", "status", "tasks"].includes(hash)
        ? hash
        : "map";
    for (const name of ["map", "docs", "reader", "status", "tasks"])
      $(`${name}-view`).hidden = name !== view;
    document
      .querySelectorAll("[data-nav]")
      .forEach((link) =>
        link.classList.toggle(
          "active",
          link.dataset.nav === (isDoc ? "docs" : view),
        ),
      );
    $("breadcrumb").textContent = {
      map: "設計マップ",
      docs: "ドキュメント",
      reader: "ドキュメント",
      status: "稼働状態",
      tasks: "開発タスク",
    }[view];
    state.render++;
    window.scrollTo(0, 0);
    try {
      if (isDoc) {
        const params = new URLSearchParams(hash);
        await readDoc(params.get("doc"), params.get("section"));
      }
      if (view === "status") await refreshStatus();
      if (view === "tasks") await window.akariTasks.open(state.documents, api, go);
    } catch (err) {
      error(err.message);
    }
  }
  function headingID(label) {
    return label
      .toLocaleLowerCase()
      .replace(/[^\p{L}\p{N}\s_-]/gu, "")
      .replace(/\s/g, "-");
  }
  async function readDoc(path, section) {
    const doc = lookup(path),
      version = state.render;
    if (!doc) {
      $("article").replaceChildren(text("h1", "ドキュメントが見つかりません"));
      $("toc").replaceChildren();
      $("related").replaceChildren();
      return;
    }
    $("doc-path").textContent = doc.path;
    $("doc-edit").disabled = !doc.revision;
    document.title = `${shortTitle(doc)} — Akari`;
    $("article").innerHTML = DOMPurify.sanitize(marked.parse(doc.content), {
      USE_PROFILES: { html: true },
      FORBID_TAGS: ["style", "form", "input", "iframe"],
      FORBID_ATTR: ["style"],
    });
    $("toc").replaceChildren();
    $("related").replaceChildren();
    const usedIDs = new Map();
    $("article")
      .querySelectorAll("h1,h2,h3,h4")
      .forEach((heading) => {
        const base = headingID(heading.textContent),
          count = usedIDs.get(base) || 0;
        usedIDs.set(base, count + 1);
        heading.id = count ? `${base}-${count}` : base;
        if (heading.tagName !== "H1") {
          const a = text(
            "a",
            heading.textContent,
            heading.tagName !== "H2" ? "sub" : "",
          );
          a.href = go(doc.path, heading.id);
          $("toc").append(a);
        }
      });
    $("article")
      .querySelectorAll("a")
      .forEach((a) => {
        const raw = a.getAttribute("href") || "",
          target = resolve(doc.path, raw);
        if (target && lookup(target.path))
          a.href = go(target.path, target.section);
        else if (/^https?:\/\//i.test(raw)) {
          a.target = "_blank";
          a.rel = "noopener noreferrer";
        } else {
          a.removeAttribute("href");
          a.title = "この資料はコンソールに含まれていません";
        }
      });
    const related = new Set(
      state.links
        .filter((link) => link.from === doc.path)
        .map((link) => link.to),
    );
    for (const path of related) {
      const a = text("a", shortTitle(lookup(path)));
      a.href = go(path);
      $("related").append(a);
    }
    const blocks = [
      ...$("article").querySelectorAll("pre code.language-mermaid"),
    ];
    for (const [index, block] of blocks.entries()) {
      const source = block.textContent;
      const panel = text("div", "", "diagram"),
        header = text("div", "", "diagram-header");
      header.append(text("span", "設計図 / Mermaid"));
      const controls = document.createElement("div");
      const smaller = text("button", "−"),
        larger = text("button", "＋");
      smaller.setAttribute("aria-label", "図を縮小");
      larger.setAttribute("aria-label", "図を拡大");
      controls.append(smaller, larger);
      header.append(controls);
      const body = text("div", "", "diagram-body"),
        details = document.createElement("details");
      details.append(text("summary", "図の原文を確認"), text("pre", source));
      panel.append(header, body, details);
      block.parentElement.replaceWith(panel);
      try {
        const result = await mermaid.render(
          `diagram-${version}-${index}`,
          source,
        );
        if (version !== state.render) return;
        body.innerHTML = DOMPurify.sanitize(result.svg, {
          USE_PROFILES: { svg: true, svgFilters: true },
        });
        const diagram = body.querySelector("svg");
        let scale = 1;
        const width = Number(diagram.getAttribute("viewBox").split(/\s+/)[2]);
        diagram.style.width = `${width}px`;
        const zoom = (change) => {
          scale = Math.max(0.5, Math.min(3, scale + change));
          diagram.style.width = `${width * scale}px`;
        };
        smaller.addEventListener("click", () => zoom(-0.25));
        larger.addEventListener("click", () => zoom(0.25));
      } catch {
        body.append(
          text("p", "図を描画できませんでした。原文を確認してください。"),
        );
        details.open = true;
      }
    }
    if (section && version === state.render) {
      const target = [...$("article").querySelectorAll("[id]")].find(
        (el) => el.id === section || el.id === headingID(section),
      );
      target?.scrollIntoView({ block: "start" });
    }
  }
  async function refreshStatus() {
    const value = await api("/api/status");
    $("connection").textContent = "接続中";
    $("runtime-stats").replaceChildren();
    for (const [label, count] of [
      ["採用版", value.revision],
      ["保存Sequence", value.sequence],
      ["保存Schema", value.schema],
      ["承認待ち", value.awaitingApproval],
    ]) {
      const card = text("div", "", "runtime-stat");
      card.append(text("p", label), text("strong", count));
      $("runtime-stats").append(card);
    }
    $("outbox").replaceChildren();
    for (const [stage, label] of [
      ["queued", "実行待ち"],
      ["running", "実行中"],
      ["succeeded", "完了"],
      ["not_executed", "未実行・取消"],
      ["unknown", "結果不明"],
    ]) {
      const card = text("div", label);
      card.append(text("strong", value.outbox[stage] || 0));
      $("outbox").append(card);
    }
    $("updated-at").textContent =
      `更新 ${new Date().toLocaleTimeString("ja-JP")}`;
    const dl = document.createElement("dl");
    for (const [label, count] of [
      ["共有文脈の項目", value.sharedItems],
      ["作業記憶", value.workingMemories],
      ["関心の項目", value.interests],
    ])
      dl.append(text("dt", label), text("dd", count));
    $("inner-stats").replaceChildren(dl);
  }
  $("login-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    $("login-error").textContent = "";
    try {
      await api("/api/session", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token: $("code").value.trim() }),
      });
      await initialize();
    } catch (err) {
      $("login-error").textContent = err.message;
    }
  });
  $("logout").addEventListener("click", async () => {
    try {
      await api("/api/session", { method: "DELETE" });
      showLogin(false);
    } catch (err) {
      error(err.message);
    }
  });
  $("search").addEventListener("input", renderLibrary);
  $("docs-refresh").addEventListener("click", () => initialize().catch((err) => error(err.message)));
  function editDocument(doc) {
    documentRevision = doc?.revision || state.documentRevision || "";
    if (!documentRevision) { error("Gitリポジトリの資料を更新してから編集してください。"); return; }
    $("doc-edit-path").value = doc?.path || "";
    $("doc-edit-path").readOnly = !!doc;
    $("doc-content").value = doc?.content || "# 新しい資料\n\n";
    $("doc-dialog-title").textContent = doc ? "Markdownを編集" : "資料を追加";
    $("doc-edit-revision").textContent = `Git ${documentRevision.slice(0, 12)}`;
    $("doc-form-error").hidden = true;
    $("doc-dialog").showModal();
  }
  $("doc-edit").addEventListener("click", () => editDocument(lookup(new URLSearchParams(location.hash.slice(1)).get("doc"))));
  $("doc-new").addEventListener("click", () => editDocument(null));
  $("doc-cancel").addEventListener("click", () => { if (!documentSaving) $("doc-dialog").close(); });
  $("doc-dialog").addEventListener("cancel", (event) => { if (documentSaving) event.preventDefault(); });
  $("doc-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    if (documentSaving) return;
    documentSaving = true;
    $("doc-cancel").disabled = true;
    $("doc-save").disabled = true;
    $("doc-form-error").hidden = true;
    try {
      const updated = await api("/api/docs", { method: "PUT", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: $("doc-edit-path").value.trim(), content: $("doc-content").value, revision: documentRevision }) });
      $("doc-dialog").close();
      await initialize();
      location.hash = go(updated.path);
    } catch (err) {
      $("doc-form-error").textContent = err.message;
      $("doc-form-error").hidden = false;
    } finally {
      documentSaving = false;
      $("doc-cancel").disabled = false;
      $("doc-save").disabled = false;
    }
  });
  document.querySelectorAll("[data-group]").forEach((button) =>
    button.addEventListener("click", () => {
      state.group = button.dataset.group;
      document
        .querySelectorAll("[data-group]")
        .forEach((b) => b.classList.toggle("active", b === button));
      renderMap();
    }),
  );
  document.querySelectorAll("[data-filter]").forEach((button) =>
    button.addEventListener("click", () => {
      state.filter = button.dataset.filter;
      document
        .querySelectorAll("[data-filter]")
        .forEach((b) => b.classList.toggle("active", b === button));
      renderLibrary();
    }),
  );
  function zoom(change) {
    state.zoom = change
      ? Math.max(75, Math.min(200, state.zoom + change))
      : 100;
    $("graph").style.width = `${state.zoom}%`;
    $("zoom-reset").textContent = `${state.zoom}%`;
  }
  $("zoom-out").addEventListener("click", () => zoom(-25));
  $("zoom-in").addEventListener("click", () => zoom(25));
  $("zoom-reset").addEventListener("click", () => zoom(0));
  $("refresh").addEventListener("click", () =>
    refreshStatus().catch((err) => error(err.message)),
  );
  window.addEventListener("hashchange", route);
  initialize().catch(() => showLogin());
})();
