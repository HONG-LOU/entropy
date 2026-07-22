import test from "node:test";
import assert from "node:assert/strict";
import { broadcastTransaction } from "../src/api.js";

test("节点已知交易时广播按成功处理", async (context) => {
  const originalFetch = globalThis.fetch;
  context.after(() => { globalThis.fetch = originalFetch; });
  globalThis.fetch = async () => new Response(JSON.stringify({ error: "transaction already known" }), {
    status: 409,
    headers: { "Content-Type": "application/json" },
  });
  const result = await broadcastTransaction({ id: "a".repeat(64) });
  assert.equal(result.transaction_id, "a".repeat(64));
  assert.equal(result.already_known, true);
});
