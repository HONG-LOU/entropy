import assert from "node:assert/strict";
import test from "node:test";

import { transactionKind } from "../src/transaction-filter.js";

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
