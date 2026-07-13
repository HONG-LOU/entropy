import {
  ArrowUpRight,
  Copy,
  Cpu,
  Pickaxe,
  Play,
  Plus,
  Send,
  Square,
  SquareDot,
  createIcons,
} from "lucide";
import "./style.css";

const appIcons = { ArrowUpRight, Copy, Cpu, Pickaxe, Play, Plus, Send, Square, SquareDot };

const mockDashboard = {
  name: "Entropy",
  symbol: "ENT",
  address: "ent1c4f1a78d3420f6eb6cc89f29bd54bc144b0ea8d4a77cd41",
  confirmed_balance: "0.12683918",
  spendable_balance: "0.12683918",
  height: 2,
  tip_hash: "000002afcab83521ec7111b5432e21779ad33ce9c7cdf863b7fa5285fe183a28",
  difficulty: 22,
  pending_count: 1,
  peer_count: 1,
  configured_peer_count: 2,
  peers: [
    { url: "http://192.168.1.20:47821", online: true },
    { url: "http://10.0.0.8:47821", online: false },
  ],
  mining: false,
  listen_address: "[::]:47821",
  issued: "0.12683918",
  max_supply: "2000000",
  target_block_seconds: 10,
  emission_blocks: 31536000,
  next_subsidy: "0.06341959",
  last_error: "",
  recent_blocks: [
    { height: 2, hash: "000002afcab83521ec7111b5432e21779ad33ce9c7cdf863b7fa5285fe183a28", timestamp: 1783923020, transactions: 2, difficulty: 22 },
    { height: 1, hash: "00000cb251e9a90c2269228f97adb4040fec174ebe2aa5f3215eb2f5a25aa800", timestamp: 1783923010, transactions: 1, difficulty: 22 },
    { height: 0, hash: "c7108201a36db97765911f4362c4af3f24294e5031e17d52f1115f7b7712e435", timestamp: 1783900800, transactions: 0, difficulty: 0 },
  ],
};

const elements = Object.fromEntries(
  [
    "confirmed-balance", "spendable-balance", "wallet-address", "height", "peer-count",
    "pending-count", "difficulty", "target-seconds", "listen-address", "issued", "max-supply",
    "supply-progress", "tip-hash", "blocks-body", "peer-list", "peer-badge", "mining-label",
    "mining-indicator", "toggle-mining", "mine-once", "status-dot", "node-state-label", "toast",
    "next-reward",
  ].map((id) => [id, document.getElementById(id)]),
);

let dashboard = mockDashboard;
let toastTimer;

function backend() {
  return window.go?.main?.App;
}

async function invoke(method, ...args) {
  const api = backend();
  if (api?.[method]) return api[method](...args);
  if (method === "GetDashboard") return mockDashboard;
  await new Promise((resolve) => setTimeout(resolve, 350));
  return { message: "Preview action completed" };
}

function shortHash(value, length = 12) {
  if (!value || value.length <= length) return value || "--";
  return `${value.slice(0, length)}...`;
}

function formatTimestamp(timestamp) {
  return new Intl.DateTimeFormat("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date(timestamp * 1000));
}

function render(data) {
  dashboard = data;
  elements["confirmed-balance"].textContent = data.confirmed_balance;
  elements["spendable-balance"].textContent = data.spendable_balance;
  elements["wallet-address"].textContent = data.address;
  elements.height.textContent = data.height.toLocaleString();
  elements["peer-count"].textContent = data.peer_count;
  elements["pending-count"].textContent = data.pending_count;
  elements.difficulty.textContent = data.difficulty;
  elements["target-seconds"].textContent = data.target_block_seconds;
  elements["listen-address"].textContent = data.listen_address || "Starting";
  elements.issued.textContent = data.issued;
  elements["max-supply"].textContent = Number(data.max_supply).toLocaleString();
  elements["next-reward"].textContent = data.next_subsidy;
  elements["tip-hash"].textContent = shortHash(data.tip_hash, 18);
  elements["tip-hash"].title = data.tip_hash;
  elements["peer-badge"].textContent = `${data.peer_count}/${data.configured_peer_count}`;
  elements["status-dot"].classList.toggle("error", Boolean(data.last_error));
  elements["node-state-label"].textContent = data.last_error ? "Node warning" : "Node active";

  const supplyPercent = Math.min(100, (Number(data.issued) / Number(data.max_supply)) * 100);
  elements["supply-progress"].style.width = `${Math.max(supplyPercent, data.height > 0 ? 0.35 : 0)}%`;

  elements["blocks-body"].innerHTML = data.recent_blocks.map((block) => `
    <tr>
      <td><strong>#${block.height.toLocaleString()}</strong></td>
      <td><code title="${block.hash}">${shortHash(block.hash)}</code></td>
      <td>${formatTimestamp(block.timestamp)}</td>
      <td>${block.transactions}</td>
      <td>${block.difficulty}</td>
    </tr>
  `).join("");

  elements["peer-list"].replaceChildren();
  if (data.peers.length === 0) {
    const empty = document.createElement("li");
    empty.className = "empty-row";
    empty.textContent = "No peers connected";
    elements["peer-list"].append(empty);
  } else {
    for (const peer of data.peers) {
      const row = document.createElement("li");
      const dot = document.createElement("span");
      const value = document.createElement("code");
      dot.className = `peer-dot${peer.online ? "" : " offline"}`;
      value.textContent = peer.url;
      row.append(dot, value);
      elements["peer-list"].append(row);
    }
  }

  elements["mining-label"].textContent = data.mining ? "Mining" : "Stopped";
  elements["mining-indicator"].classList.toggle("active", data.mining);
  elements["toggle-mining"].classList.toggle("danger", data.mining);
  elements["toggle-mining"].innerHTML = data.mining
    ? '<i data-lucide="square"></i><span>Stop mining</span>'
    : '<i data-lucide="play"></i><span>Start mining</span>';
  elements["mine-once"].disabled = data.mining;
  createIcons({ icons: appIcons });
}

function showToast(message, type = "success") {
  clearTimeout(toastTimer);
  elements.toast.textContent = message;
  elements.toast.className = `toast visible ${type}`;
  toastTimer = setTimeout(() => { elements.toast.className = "toast"; }, 3200);
}

function errorMessage(error) {
  if (typeof error === "string") return error;
  return error?.message || "Action failed";
}

async function refresh() {
  try {
    render(await invoke("GetDashboard"));
  } catch (error) {
    elements["status-dot"].classList.add("error");
    elements["node-state-label"].textContent = "Node offline";
    showToast(errorMessage(error), "error");
  }
}

document.getElementById("copy-address").addEventListener("click", async () => {
  await navigator.clipboard.writeText(dashboard.address);
  showToast("Address copied");
});

document.getElementById("send-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const button = event.currentTarget.querySelector("button[type=submit]");
  button.disabled = true;
  try {
    const result = await invoke(
      "SendTransaction",
      document.getElementById("send-to").value.trim(),
      document.getElementById("send-amount").value.trim(),
      document.getElementById("send-fee").value.trim(),
    );
    showToast(`${result.message}: ${shortHash(result.id)}`);
    document.getElementById("send-amount").value = "";
    await refresh();
  } catch (error) {
    showToast(errorMessage(error), "error");
  } finally {
    button.disabled = false;
  }
});

elements["toggle-mining"].addEventListener("click", async () => {
  elements["toggle-mining"].disabled = true;
  try {
    const result = await invoke("SetMining", !dashboard.mining);
    showToast(result.message);
    await refresh();
  } catch (error) {
    showToast(errorMessage(error), "error");
  } finally {
    elements["toggle-mining"].disabled = false;
  }
});

elements["mine-once"].addEventListener("click", async () => {
  elements["mine-once"].disabled = true;
  try {
    const result = await invoke("MineOneBlock");
    showToast(result.message);
    await refresh();
  } catch (error) {
    showToast(errorMessage(error), "error");
  } finally {
    elements["mine-once"].disabled = false;
  }
});

document.getElementById("peer-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const input = document.getElementById("peer-url");
  try {
    const result = await invoke("AddPeer", input.value.trim());
    showToast(result.message);
    input.value = "";
    await refresh();
  } catch (error) {
    showToast(errorMessage(error), "error");
  }
});

createIcons({ icons: appIcons });
refresh();
setInterval(refresh, 1000);
