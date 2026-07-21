export const TRANSACTION_FILTERS = new Set(["all", "received", "sent", "mining"]);

export function transactionKind(transaction) {
  if (transaction?.coinbase) return "mining";
  if (!/^0(?:\.0+)?$/.test(String(transaction?.sent ?? "0"))) return "sent";
  return "received";
}

export function filterTransactions(transactions, filter) {
  if (filter === "all") return transactions;
  if (!TRANSACTION_FILTERS.has(filter)) return transactions;
  return transactions.filter((transaction) => transactionKind(transaction) === filter);
}
