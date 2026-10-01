import { execFile, spawn, type ChildProcess } from "node:child_process";
import { randomBytes } from "node:crypto";
import { chmod, mkdir, mkdtemp, readFile, realpath, rename, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { promisify } from "node:util";

const exec = promisify(execFile);

export interface Post {
  id: string;
  slug: string;
  title: string;
  /** file is the post's index.md, relative to the site's root. */
  file: string;
}

interface PostSpec {
  slug: string;
  title: string;
  body: string;
  status?: "published" | "draft";
  date: string;
  tags?: string[];
}

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ";

/**
 * newId makes a ULID. Every site gets ids of its own, so a server can be told
 * apart from the one another test started on the same port a moment earlier.
 */
function newId(): string {
  let time = Date.now();
  let id = "";
  for (let i = 0; i < 10; i++) {
    id = crockford[time % 32] + id;
    time = Math.floor(time / 32);
  }
  for (const byte of randomBytes(16)) id += crockford[byte % 32];
  return id;
}

const day = (n: number) => `2026-01-${String(n).padStart(2, "0")}T09:00:00Z`;

/**
 * The posts every site starts with: enough garden notes to page through, two
 * garden drafts, and a post for each test that changes one of its own.
 */
const posts: PostSpec[] = [
  ...Array.from({ length: 24 }, (_, i): PostSpec => {
    const n = String(i + 1).padStart(2, "0");
    return {
      slug: `garden-note-${n}`,
      title: `Garden note ${n}`,
      body: `Notes from the garden, entry ${n}.`,
      date: day(i + 1),
      tags: ["garden"],
    };
  }),
  { slug: "garden-draft-a", title: "Garden draft A", body: "Not ready.", status: "draft", date: day(25), tags: ["garden"] },
  { slug: "garden-draft-b", title: "Garden draft B", body: "Not ready.", status: "draft", date: day(26), tags: ["garden"] },
  {
    slug: "three-paragraphs",
    title: "Three paragraphs",
    body: "Alpha paragraph.\n\nBravo paragraph.\n\nCharlie paragraph.",
    date: "2025-12-01T09:00:00Z",
  },
  { slug: "slow-save", title: "Slow save", body: "Written before the save.", date: "2025-12-02T09:00:00Z" },
  { slug: "coast", title: "Pictures from the coast", body: "A day by the sea.", date: "2025-12-03T09:00:00Z" },
  { slug: "old-news-one", title: "Old news one", body: "Yesterday.", date: "2025-12-04T09:00:00Z" },
  { slug: "old-news-two", title: "Old news two", body: "The day before.", date: "2025-12-05T09:00:00Z" },
  { slug: "ready", title: "Ready to publish", body: "First words.", date: "2025-12-06T09:00:00Z" },
];

function frontMatter(spec: PostSpec, id: string): string {
  const status = spec.status ?? "published";
  const lines = [
    "---",
    `id: ${id}`,
    `title: ${spec.title}`,
    `slug: ${spec.slug}`,
    `status: ${status}`,
    `created_at: ${spec.date}`,
  ];
  if (status === "published") lines.push(`published_at: ${spec.date}`);
  if (spec.tags?.length) lines.push("tags:", ...spec.tags.map((tag) => `  - ${tag}`));
  lines.push("---", "", spec.body, "");
  return lines.join("\n");
}

/**
 * environment is what kite and git run with: no account or config overrides
 * from the machine running the tests, and none of its git configuration, so
 * a signing key or a global hooks path there cannot change a result here.
 */
function environment(base: string): NodeJS.ProcessEnv {
  const env: NodeJS.ProcessEnv = {};
  for (const [key, value] of Object.entries(process.env)) {
    if (!key.startsWith("KITE_") && !key.startsWith("GIT_")) env[key] = value;
  }
  env.GIT_CONFIG_GLOBAL = path.join(base, "gitconfig");
  env.GIT_CONFIG_NOSYSTEM = "1";
  return env;
}

const pause = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

/** deadIndex is where no index answers, as the site's own until a test serves one. */
export const deadIndex = "http://127.0.0.1:9/index.json";

/**
 * Site is one throwaway project served by a real kite binary: a git
 * repository with a bare repository beside it as its remote, in a temporary
 * directory of its own.
 */
export class Site {
  readonly posts: Record<string, Post> = {};
  /** log is everything the server printed, for a test that failed. */
  log = "";
  url = "";

  private server?: ChildProcess;
  private readonly env: NodeJS.ProcessEnv;

  private constructor(
    /** base holds the project, its remote and the git configuration. */
    readonly base: string,
    readonly root: string,
    readonly remote: string,
  ) {
    this.env = environment(base);
  }

  /** create makes the site and serves it; watch=false leaves the server blind to changes on disk. */
  static async create({ watch = true }: { watch?: boolean } = {}): Promise<Site> {
    const bin = process.env.KITE_BIN;
    if (!bin) throw new Error("KITE_BIN is not set; the global setup should have set it");

    // Real paths: macOS hands out temporary directories through a symlink,
    // and git and kite would each see a different name for the same folder.
    const base = await realpath(await mkdtemp(path.join(tmpdir(), "kite-e2e-")));
    const site = new Site(base, path.join(base, "site"), path.join(base, "remote.git"));
    try {
      await site.init(bin);
      await site.start(bin, watch);
    } catch (err) {
      await site.stop();
      await site.remove();
      throw err;
    }
    return site;
  }

  private async init(bin: string) {
    await writeFile(path.join(this.base, "gitconfig"), "");
    await mkdir(this.root);
    await exec(bin, ["init", ".", "--yes", "--workflow=false", "--title", "E2E Site", "--author", "E2E Author"], {
      cwd: this.root,
      env: this.env,
    });
    // No test reaches the real index of themes and plugins: one that wants
    // an index serves its own here.
    const config = path.join(this.root, "kite.yaml");
    await writeFile(config, `${(await readFile(config, "utf8")).trimEnd()}\napps:\n  index: ${deadIndex}\n`);

    for (const spec of posts) {
      const id = newId();
      const file = `content/posts/${spec.slug}/index.md`;
      await mkdir(path.join(this.root, path.dirname(file)), { recursive: true });
      await writeFile(path.join(this.root, file), frontMatter(spec, id));
      this.posts[spec.slug] = { id, slug: spec.slug, title: spec.title, file };
    }

    await this.git("init", "-q", "-b", "main");
    await this.git("config", "user.name", "Kite E2E");
    await this.git("config", "user.email", "e2e@example.invalid");
    await this.git("add", "-A");
    await this.git("commit", "-q", "-m", "initial");
    await exec("git", ["init", "-q", "--bare", "-b", "main", this.remote], { env: this.env });
    await this.git("remote", "add", "origin", this.remote);
    await this.git("push", "-q", "-u", "origin", "main");
  }

  /**
   * start runs kite serve on a port the operating system picks. The port is
   * free when kite asks for it but can be taken before kite listens on it,
   * by the server of a test running beside this one, so a start that loses
   * that race is tried again.
   */
  private async start(bin: string, watch: boolean) {
    for (let attempt = 1; ; attempt++) {
      const from = this.log.length;
      try {
        await this.listen(bin, watch);
        return;
      } catch (err) {
        if (attempt === 3 || !/address already in use/.test(this.log.slice(from))) throw err;
      }
    }
  }

  private async listen(bin: string, watch: boolean) {
    const flags = ["--admin", "--write", "--drafts", "--port", "0", "--live-reload=false", `--watch=${watch}`];
    const server = spawn(bin, ["serve", ...flags], {
      cwd: this.root,
      env: this.env,
      stdio: ["ignore", "pipe", "pipe"],
    });
    this.server = server;

    let printed = "";
    const url = await new Promise<string>((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error(`kite serve printed no address:\n${this.log}`)), 15_000);
      server.stdout?.on("data", (chunk: Buffer) => {
        this.log += chunk.toString();
        printed += chunk.toString();
        const found = /(http:\/\/127\.0\.0\.1:\d+)\s/.exec(printed);
        if (found) {
          clearTimeout(timer);
          resolve(found[1]);
        }
      });
      server.stderr?.on("data", (chunk: Buffer) => {
        this.log += chunk.toString();
      });
      server.once("exit", () => {
        clearTimeout(timer);
        reject(new Error(`kite serve exited:\n${this.log}`));
      });
    });

    // The address is printed before the server listens on it, and until it
    // does another server may be answering there: only one of this site's
    // own posts says the server is this one.
    const mine = `${url}/api/v1/contents/${Object.values(this.posts)[0].id}`;
    const deadline = Date.now() + 15_000;
    while (Date.now() < deadline) {
      if (server.exitCode !== null || server.signalCode !== null) throw new Error(`kite serve exited:\n${this.log}`);
      try {
        if ((await fetch(mine)).ok) {
          this.url = url;
          return;
        }
      } catch {
        // Not listening yet.
      }
      await pause(50);
    }
    throw new Error(`kite serve never answered at ${url}:\n${this.log}`);
  }

  async stop() {
    const server = this.server;
    if (!server || server.exitCode !== null || server.signalCode !== null) return;
    const exited = new Promise((resolve) => server.once("exit", resolve));
    server.kill("SIGTERM");
    const timer = setTimeout(() => server.kill("SIGKILL"), 5_000);
    await exited;
    clearTimeout(timer);
  }

  async remove() {
    await rm(this.base, { recursive: true, force: true, maxRetries: 3 });
  }

  /** git runs git in the project, with the tests' own configuration. */
  async git(...args: string[]): Promise<string> {
    const { stdout } = await exec("git", args, { cwd: this.root, env: this.env });
    return stdout.trim();
  }

  /** hook installs a git hook in the project's repository. */
  async hook(name: string, script: string) {
    const file = path.join(this.root, ".git", "hooks", name);
    await writeFile(file, script);
    await chmod(file, 0o755);
  }

  /** remoteHead is the commit the remote's main branch points at. */
  async remoteHead(): Promise<string> {
    const { stdout } = await exec("git", ["--git-dir", this.remote, "rev-parse", "main"], { env: this.env });
    return stdout.trim();
  }

  read(file: string): Promise<string> {
    return readFile(path.join(this.root, file), "utf8");
  }

  /**
   * write replaces a file whole, as an editor saving it does, so the server's
   * watcher never reads it half written.
   */
  async write(file: string, text: string) {
    const draft = path.join(this.base, `draft-${randomBytes(4).toString("hex")}`);
    await writeFile(draft, text);
    await rename(draft, path.join(this.root, file));
  }

  /** api calls the server as a client of its own, with no browser session. */
  api(route: string, init?: RequestInit): Promise<Response> {
    return fetch(`${this.url}/api/v1${route}`, init);
  }

  /** textOf reads the index.md of a post that was made in the studio. */
  async textOf(id: string): Promise<string> {
    const response = await this.api(`/contents/${id}`);
    if (!response.ok) throw new Error(`GET /contents/${id}: ${response.status}`);
    const { locator } = (await response.json()) as { locator: string };
    return this.read(`${locator}/index.md`);
  }

  /**
   * edit changes a post's file as another editor would, and waits until the
   * server has read it back: its index follows the disk through a watcher,
   * a moment behind, and a conflict is reported with what the index holds.
   */
  async edit(post: Post, change: (text: string) => string) {
    await this.change(post.file, `/contents/${post.id}`, change);
  }

  /** editConfig does the same for kite.yaml, which the settings are read from. */
  async editConfig(change: (text: string) => string) {
    await this.change("kite.yaml", "/settings", change);
  }

  /** change rewrites a file and waits until the version the API reports at route is a new one. */
  private async change(file: string, route: string, change: (text: string) => string) {
    const version = async () => {
      const response = await this.api(route);
      return response.ok ? (response.headers.get("ETag") ?? "") : "";
    };
    const before = await version();
    await this.write(file, change(await this.read(file)));
    const deadline = Date.now() + 10_000;
    let now = before;
    while (now === before || now === "") {
      if (Date.now() > deadline) throw new Error(`the server never read the new ${file}`);
      await pause(50);
      now = await version();
    }
  }
}
