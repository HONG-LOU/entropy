import assert from "node:assert/strict";
import test from "node:test";

import { filterTransactions, transactionKind } from "../src/transaction-filter.js";

const history = [
  { id: "received", received: "2.5", sent: "0", coinbase: false },
  { id: "sent", received: "0", sent: "1.25", coinbase: false },
  { id: "mining", received: "50", sent: "0", coinbase: true },
];

test("transactionKind gives mining precedence over transfer amounts", () => {
  assert.equal(transactionKind(history[0]), "received");
  assert.equal(transactionKind(history[1]), "sent");
  assert.equal(transactionKind(history[2]), "mining");
  assert.equal(transactionKind({ sent: "0.00000000" }), "received");
});

test("filterTransactions supports all transaction filters", () => {
  assert.deepEqual(filterTransactions(history, "all"), history);
  assert.deepEqual(filterTransactions(history, "received").map(({ id }) => id), ["received"]);
  assert.deepEqual(filterTransactions(history, "sent").map(({ id }) => id), ["sent"]);
  assert.deepEqual(filterTransactions(history, "mining").map(({ id }) => id), ["mining"]);
});

test("unknown filters fall back to the complete history", () => {
  assert.deepEqual(filterTransactions(history, "unknown"), history);
});
