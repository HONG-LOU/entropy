export const FALLBACK_RELEASE_URL = "https://github.com/HONG-LOU/entcoin/releases/tag/v1.0.6";

const en = {
  "meta.title": "Entcoin — Run the network. Verify everything.",
  "meta.description": "Entcoin is a compact proof-of-work network with a local wallet, validating full node, SQLite ledger, peer relay, and optional miner in one desktop application.",
  "skip.content": "Skip to content",
  "brand.tagline": "Proof-of-work, held locally",
  "nav.network": "Network",
  "nav.technology": "Technology",
  "nav.economics": "Economics",
  "nav.download": "Download",
  "nav.docs": "Docs",
  "nav.language": "中文",
  "nav.open": "Open navigation",
  "nav.close": "Close navigation",
  "hero.eyebrow": "Entcoin mainnet / protocol v1",
  "hero.title": "Run the network. Verify everything.",
  "hero.body": "A compact proof-of-work network in one desktop application: wallet, validating full node, SQLite ledger, peer relay, and optional miner.",
  "hero.download": "Run a node",
  "hero.protocol": "Read the protocol",
  "hero.signal": "Public archive seed",
  "hero.online": "Online",
  "hero.unavailable": "Status unavailable",
  "metric.height": "Live height",
  "metric.height.pending": "Connecting",
  "metric.spacing": "Block target",
  "metric.supply": "Maximum ENT",
  "metric.premine": "Premine",
  "metric.release": "Current release",
  "product.eyebrow": "01 / The node",
  "product.title": "Everything local. Nothing hidden behind an account.",
  "product.body": "Opening Entcoin starts a real validating node. Private keys stay in the local node process, the ledger is stored in SQLite, and every accepted block is checked before state changes.",
  "product.window.status": "Node active",
  "product.window.overview": "Overview",
  "product.window.transactions": "Transactions",
  "product.window.network": "Network",
  "product.window.online": "Online",
  "product.window.synced": "Synced and relaying",
  "product.window.wallet": "Wallet",
  "product.window.diagnostics": "Diagnostics",
  "product.window.confirmed": "Confirmed",
  "product.window.height": "Chain height",
  "product.window.storage": "Storage",
  "product.window.archive": "Archive",
  "product.point.wallet.title": "Self-custodied wallet",
  "product.point.wallet.body": "24-word recovery and an encrypted portable backup, protected locally.",
  "product.point.validation.title": "Full validation",
  "product.point.validation.body": "UTXOs, signatures, proof of work, issuance, maturity, and reorg rules.",
  "product.point.process.title": "One process",
  "product.point.process.body": "No hosted login, browser wallet, external database, or coordinator.",
  "technology.eyebrow": "02 / Verification path",
  "technology.title": "From peer message to committed ledger.",
  "technology.body": "Headers arrive first. Candidate work is compared locally. Bodies are fetched only for a viable chain, fully validated, and committed atomically with undo data.",
  "technology.discover.title": "Discover",
  "technology.discover.body": "Built-in HTTPS bootstrap seeds and local network discovery connect the node automatically.",
  "technology.validate.title": "Validate",
  "technology.validate.body": "Headers, timestamps, difficulty, work, ownership, and signatures.",
  "technology.commit.title": "Commit",
  "technology.commit.body": "One SQLite transaction with WAL, full synchronization, and undo records.",
  "technology.relay.title": "Relay",
  "technology.relay.body": "Accepted blocks and transactions move over bounded HTTP and WebSocket paths.",
  "technology.architecture": "Explore the architecture",
  "economics.eyebrow": "03 / Monetary rules",
  "economics.title": "A small, explicit issuance schedule.",
  "economics.body": "No premine, no account balances, and no floating-point supply math. Rewards are determined by block height and sum to an exact terminal amount.",
  "economics.max": "Maximum supply",
  "economics.units": "Atomic units / ENT",
  "economics.spacing": "Target spacing",
  "economics.maturity": "Coinbase maturity",
  "economics.choice": "Fork choice",
  "economics.choice.value": "Cumulative work",
  "economics.blocks": "reward-bearing blocks",
  "economics.schedule": "across an approximately ten-year target schedule",
  "economics.progress": "Issuance progress is derived from the validated chain, never a hosted counter.",
  "download.eyebrow": "04 / Join",
  "download.title": "Choose your node.",
  "download.body": "Desktop builds include the wallet and operational interface. The headless CLI runs the same validation, ledger, and peer-to-peer implementation.",
  "download.windows.platform": "Windows 10/11 · x64",
  "download.windows.title": "Desktop node",
  "download.windows.body": "Installer or portable build with wallet, mining, peers, history, and diagnostics.",
  "download.windows.action": "Windows downloads",
  "download.ubuntu.platform": "Ubuntu 24.04+ · amd64",
  "download.ubuntu.title": "Desktop node",
  "download.ubuntu.body": "Native .deb with a Secret Service protected local wallet.",
  "download.ubuntu.action": "Download .deb",
  "download.cli.platform": "Linux / Windows",
  "download.cli.title": "Headless CLI",
  "download.cli.body": "Archive, pruned, wallet, or public seed deployments from one binary.",
  "download.cli.action": "CLI downloads",
  "download.checksums": "Checksums",
  "download.release": "View release notes",
  "download.boundary": "v1.0.6 is unsigned and may trigger SmartScreen. Verify the published checksums and understand their limits.",
  "infrastructure.eyebrow": "05 / Live infrastructure",
  "infrastructure.title": "Inspect the public node.",
  "infrastructure.body": "This page reads the same bounded status endpoint other nodes use. No market ticker and no fabricated activity.",
  "infrastructure.endpoint": "Open status endpoint",
  "infrastructure.protocol": "protocol",
  "infrastructure.height": "height",
  "infrastructure.tip": "tip hash",
  "infrastructure.transport": "transport",
  "infrastructure.storage": "storage",
  "infrastructure.storage.value": "archive seed",
  "source.eyebrow": "06 / Open source",
  "source.title": "The implementation is the claim.",
  "source.body": "Entcoin is MIT-licensed. Consensus, wallet, ledger, networking, tests, operations, and known limitations are public in the repository.",
  "source.repository": "Browse source",
  "source.operations": "Node operations",
  "source.roadmap": "Maturity roadmap",
  "source.security": "Security policy",
  "security.label": "Security boundary",
  "security.body": "Entcoin has not received an independent security or consensus audit. “Mainnet” identifies protocol compatibility, not production safety. ENT must not carry real-world value until appropriate independent review establishes a suitable basis.",
  "footer.description": "A compact, independent proof-of-work network.",
  "footer.github": "GitHub",
  "footer.releases": "Releases",
  "footer.protocol": "Protocol",
  "footer.operations": "Operations",
  "footer.security": "Security",
  "footer.license": "MIT License",
  "status.connected": "Connected",
  "status.unavailable": "Temporarily unavailable",
  "status.loading": "Reading the public node",
  "menu.windows.installer": "Windows installer",
  "menu.windows.portable": "Windows portable",
  "menu.ubuntu": "Ubuntu .deb",
  "menu.linux.cli": "Linux CLI",
  "menu.windows.cli": "Windows CLI",
};

const zh = {
  "meta.title": "Entcoin — 运行网络，独立验证",
  "meta.description": "Entcoin 是一个紧凑的工作量证明网络，在一个桌面应用中集成本地钱包、验证全节点、SQLite 账本、节点中继与可选挖矿。",
  "skip.content": "跳到正文",
  "brand.tagline": "工作量证明，本地掌控",
  "nav.network": "网络",
  "nav.technology": "技术",
  "nav.economics": "经济规则",
  "nav.download": "下载",
  "nav.docs": "文档",
  "nav.language": "EN",
  "nav.open": "打开导航",
  "nav.close": "关闭导航",
  "hero.eyebrow": "Entcoin 主网 / 协议 v1",
  "hero.title": "运行网络，独立验证。",
  "hero.body": "一个紧凑的工作量证明网络：钱包、验证全节点、SQLite 账本、节点中继与可选挖矿，都在同一个桌面应用中。",
  "hero.download": "运行节点",
  "hero.protocol": "阅读协议",
  "hero.signal": "公共归档种子",
  "hero.online": "在线",
  "hero.unavailable": "状态不可用",
  "metric.height": "实时高度",
  "metric.height.pending": "连接中",
  "metric.spacing": "出块目标",
  "metric.supply": "最大 ENT",
  "metric.premine": "预挖",
  "metric.release": "当前版本",
  "product.eyebrow": "01 / 节点",
  "product.title": "一切都在本地，没有隐藏账户。",
  "product.body": "打开 Entcoin 就会启动一个真正的验证节点。私钥留在本地节点进程，账本存入 SQLite，每个被接受的区块都会在状态变更前完成检查。",
  "product.window.status": "节点运行中",
  "product.window.overview": "概览",
  "product.window.transactions": "交易",
  "product.window.network": "网络",
  "product.window.online": "在线",
  "product.window.synced": "已同步并正在中继",
  "product.window.wallet": "钱包",
  "product.window.diagnostics": "诊断",
  "product.window.confirmed": "已确认",
  "product.window.height": "链高度",
  "product.window.storage": "存储",
  "product.window.archive": "归档",
  "product.point.wallet.title": "自托管钱包",
  "product.point.wallet.body": "24 词恢复短语与加密便携备份，都在本地保护。",
  "product.point.validation.title": "完整验证",
  "product.point.validation.body": "验证 UTXO、签名、工作量证明、发行、成熟期与重组规则。",
  "product.point.process.title": "单一进程",
  "product.point.process.body": "没有托管登录、浏览器钱包、外部数据库或协调者。",
  "technology.eyebrow": "02 / 验证路径",
  "technology.title": "从节点消息到已提交账本。",
  "technology.body": "先接收区块头，在本地比较候选累计工作量；仅为可行链下载区块体，完整验证后连同撤销数据原子提交。",
  "technology.discover.title": "发现",
  "technology.discover.body": "通过内置 HTTPS 引导种子与局域网发现自动连接网络。",
  "technology.validate.title": "验证",
  "technology.validate.body": "检查区块头、时间戳、难度、工作量、所有权与签名。",
  "technology.commit.title": "提交",
  "technology.commit.body": "通过 WAL、全同步和撤销记录在一个 SQLite 事务内提交。",
  "technology.relay.title": "中继",
  "technology.relay.body": "通过有界 HTTP 与 WebSocket 路径传递已接受的区块和交易。",
  "technology.architecture": "查看架构",
  "economics.eyebrow": "03 / 货币规则",
  "economics.title": "小而明确的发行计划。",
  "economics.body": "没有预挖、没有账户余额，也没有浮点供应量计算。奖励由区块高度决定，并精确累加到最终总量。",
  "economics.max": "最大供应量",
  "economics.units": "每 ENT 原子单位",
  "economics.spacing": "目标间隔",
  "economics.maturity": "Coinbase 成熟期",
  "economics.choice": "分叉选择",
  "economics.choice.value": "累计工作量",
  "economics.blocks": "个奖励区块",
  "economics.schedule": "覆盖约十年的目标发行周期",
  "economics.progress": "发行进度来自已验证的链，而不是托管计数器。",
  "download.eyebrow": "04 / 加入",
  "download.title": "选择你的节点。",
  "download.body": "桌面版本包含钱包与运维界面；无界面 CLI 使用相同的验证、账本和点对点实现。",
  "download.windows.platform": "Windows 10/11 · x64",
  "download.windows.title": "桌面节点",
  "download.windows.body": "安装版或便携版，包含钱包、挖矿、节点、历史与诊断。",
  "download.windows.action": "Windows 下载",
  "download.ubuntu.platform": "Ubuntu 24.04+ · amd64",
  "download.ubuntu.title": "桌面节点",
  "download.ubuntu.body": "原生 .deb，并通过 Secret Service 保护本地钱包。",
  "download.ubuntu.action": "下载 .deb",
  "download.cli.platform": "Linux / Windows",
  "download.cli.title": "无界面 CLI",
  "download.cli.body": "一个二进制支持归档、裁剪、钱包或公共种子部署。",
  "download.cli.action": "CLI 下载",
  "download.checksums": "校验值",
  "download.release": "查看版本说明",
  "download.boundary": "v1.0.6 尚未签名，可能触发 SmartScreen。请核对发布的 SHA-256 校验值，并了解校验值不能证明发布者身份。",
  "infrastructure.eyebrow": "05 / 实时基础设施",
  "infrastructure.title": "检查公共节点。",
  "infrastructure.body": "本页读取其他节点使用的同一个有界状态接口，没有行情计价，也没有伪造活动。",
  "infrastructure.endpoint": "打开状态接口",
  "infrastructure.protocol": "协议",
  "infrastructure.height": "高度",
  "infrastructure.tip": "链尖哈希",
  "infrastructure.transport": "传输",
  "infrastructure.storage": "存储",
  "infrastructure.storage.value": "归档种子",
  "source.eyebrow": "06 / 开源",
  "source.title": "实现本身就是主张。",
  "source.body": "Entcoin 使用 MIT 许可证。共识、钱包、账本、网络、测试、运维和已知限制都公开在仓库中。",
  "source.repository": "浏览源码",
  "source.operations": "节点运维",
  "source.roadmap": "成熟度路线图",
  "source.security": "安全策略",
  "security.label": "安全边界",
  "security.body": "Entcoin 尚未接受独立安全或共识审计。“主网”只表示协议兼容性，不代表生产安全。在适当的独立审查建立充分依据之前，ENT 不应承载现实世界价值。",
  "footer.description": "一个紧凑、独立的工作量证明网络。",
  "footer.github": "GitHub",
  "footer.releases": "发布版本",
  "footer.protocol": "协议",
  "footer.operations": "运维",
  "footer.security": "安全",
  "footer.license": "MIT 许可证",
  "status.connected": "已连接",
  "status.unavailable": "暂时不可用",
  "status.loading": "正在读取公共节点",
  "menu.windows.installer": "Windows 安装版",
  "menu.windows.portable": "Windows 便携版",
  "menu.ubuntu": "Ubuntu .deb",
  "menu.linux.cli": "Linux CLI",
  "menu.windows.cli": "Windows CLI",
};

export const translations = Object.freeze({
  en: Object.freeze(en),
  zh: Object.freeze(zh),
});

export function validateNodeStatus(value) {
  if (!value || typeof value !== "object") throw new TypeError("Invalid node status");
  if (value.protocol !== "entropy-mainnet-v1") throw new TypeError("Unexpected node protocol");
  if (!Number.isSafeInteger(value.height) || value.height < 0) throw new TypeError("Invalid node height");
  if (typeof value.tip_hash !== "string" || !/^[0-9a-f]{64}$/i.test(value.tip_hash)) {
    throw new TypeError("Invalid tip hash");
  }
  if (typeof value.chain_work !== "string" || !/^[0-9]{1,128}$/.test(value.chain_work)) {
    throw new TypeError("Invalid chain work");
  }

  return {
    protocol: value.protocol,
    name: value.name === "Entcoin" ? value.name : "Entcoin",
    symbol: value.symbol === "ENT" ? value.symbol : "ENT",
    height: value.height,
    tip_hash: value.tip_hash.toLowerCase(),
    chain_work: value.chain_work,
    listen_port: value.listen_port === 47_821 ? 47_821 : 47_821,
  };
}

export function formatNodeStatus(value, language = "en") {
  const status = validateNodeStatus(value);
  const locale = language === "zh" ? "zh-CN" : "en-US";
  return {
    ...status,
    height: new Intl.NumberFormat(locale).format(status.height),
    tip: `${status.tip_hash.slice(0, 12)}...${status.tip_hash.slice(-4)}`,
  };
}

export function selectReleaseAssets(value) {
  const fallback = fallbackRelease();
  if (!isStableRelease(value)) return fallback;

  const prefix = `/HONG-LOU/entcoin/releases/download/${value.tag_name}/`;
  const known = new Map();
  for (const asset of Array.isArray(value.assets) ? value.assets : []) {
    if (!asset || typeof asset.name !== "string" || typeof asset.browser_download_url !== "string") continue;
    let url;
    try {
      url = new URL(asset.browser_download_url);
    } catch {
      continue;
    }
    if (url.protocol !== "https:" || url.hostname !== "github.com" || !url.pathname.startsWith(prefix)) continue;
    known.set(asset.name, url.href);
  }

  return {
    version: value.tag_name,
    release: value.html_url,
    windowsPortable: findAsset(known, /^Entcoin\.exe$/i) ?? fallback.release,
    windowsInstaller: findAsset(known, /^entcoin-amd64-installer\.exe$/i) ?? fallback.release,
    ubuntu: findAsset(known, /^entcoin_\d+\.\d+\.\d+_amd64\.deb$/i) ?? fallback.release,
    linuxCli: findAsset(known, /^entcoin-cli-linux-amd64$/i) ?? fallback.release,
    windowsCli: findAsset(known, /^entcoin-cli\.exe$/i) ?? fallback.release,
    linuxChecksums: findAsset(known, /^SHA256SUMS-linux\.txt$/i) ?? fallback.release,
    windowsChecksums: findAsset(known, /^SHA256SUMS\.txt$/i) ?? fallback.release,
  };
}

function isStableRelease(value) {
  if (!value || typeof value !== "object" || value.draft || value.prerelease) return false;
  if (typeof value.tag_name !== "string" || !/^v1\.0\.\d+$/.test(value.tag_name)) return false;
  if (typeof value.html_url !== "string") return false;
  try {
    const url = new URL(value.html_url);
    return url.protocol === "https:" && url.hostname === "github.com" &&
      url.pathname === `/HONG-LOU/entcoin/releases/tag/${value.tag_name}`;
  } catch {
    return false;
  }
}

function findAsset(assets, pattern) {
  for (const [name, url] of assets) {
    if (pattern.test(name)) return url;
  }
  return undefined;
}

function fallbackRelease() {
  return {
    version: "v1.0.6",
    release: FALLBACK_RELEASE_URL,
    windowsPortable: FALLBACK_RELEASE_URL,
    windowsInstaller: FALLBACK_RELEASE_URL,
    ubuntu: FALLBACK_RELEASE_URL,
    linuxCli: FALLBACK_RELEASE_URL,
    windowsCli: FALLBACK_RELEASE_URL,
    linuxChecksums: FALLBACK_RELEASE_URL,
    windowsChecksums: FALLBACK_RELEASE_URL,
  };
}
