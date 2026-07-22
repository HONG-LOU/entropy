const configuredNodes = (import.meta.env?.VITE_ENTCOIN_NODES || "")
  .split(",")
  .map((node) => node.trim().replace(/\/$/, ""))
  .filter(Boolean);

export const NODES = configuredNodes.length
  ? configuredNodes
  : ["https://node.entcoin.xyz", "https://template-chat.xyz"];

async function requestJSON(url, options = {}, timeout = 10000) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeout);
  try {
    const headers = options.body ? { "Content-Type": "application/json", ...options.headers } : options.headers;
    const response = await fetch(url, { ...options, signal: controller.signal, headers });
    const body = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(body.error || `节点返回 HTTP ${response.status}`);
    return body;
  } finally {
    clearTimeout(timer);
  }
}

export async function loadWalletSnapshot(address) {
  const encoded = encodeURIComponent(address);
  const settled = await Promise.allSettled(NODES.map((node) => requestJSON(`${node}/v2/wallet/${encoded}`)));
  const available = settled.flatMap((item, index) => item.status === "fulfilled" ? [{ node: NODES[index], data: item.value }] : []);
  if (!available.length) throw new Error("暂时无法连接 Entcoin 节点");
  const primary = available[0];
  const tipsAgree = available.length > 1 && available.every(({ data }) => data.tip_hash === primary.data.tip_hash && data.chain_work === primary.data.chain_work);
  return { ...primary.data, source: primary.node, nodesOnline: available.length, tipsAgree };
}

export async function broadcastTransaction(transaction) {
  const errors = [];
  for (const node of NODES) {
    try {
      const result = await requestJSON(`${node}/v2/transactions`, { method: "POST", body: JSON.stringify(transaction) });
      return { ...result, node };
    } catch (error) {
      if (error.message.includes("already known")) {
        return { transaction_id: transaction.id, node, already_known: true };
      }
      errors.push(error.message);
    }
  }
  throw new Error(errors[0] || "交易广播失败");
}
