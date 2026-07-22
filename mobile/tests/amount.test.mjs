import test from "node:test";
import assert from "node:assert/strict";
import { formatAmount, parseAmount } from "../src/amount.js";

test("金额按八位小数精确转换", () => {
  assert.equal(parseAmount("1.00000001"), "100000001");
  assert.equal(formatAmount("100000001"), "1.00000001");
  assert.equal(formatAmount("200000000"), "2");
});

test("拒绝零、负数和过多小数", () => {
  for (const value of ["0", "-1", ".1", "1.000000001", "abc"]) {
    assert.throws(() => parseAmount(value));
  }
});
