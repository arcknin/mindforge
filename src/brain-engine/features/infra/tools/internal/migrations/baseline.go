package migrations

// GlobalMigrationLedgerBaseline pins the current immutable global catalog for
// local static audit. Runtime truth remains target-specific database evidence.
func GlobalMigrationLedgerBaseline() Ledger {
	return Ledger{
		"0001": "2625ae851aa42f59d7c071a563141fcb6ecf6c7301157786fedc1ce7e7725fdb",
		"0002": "c4fdcc68396d7bff76bc6325d3171f5d73bbb0ee273709be260ff1f30d92b7cb",
		"0003": "613d12044705182f42e228fcbbac099219e24d32e0390fed5e6e8243a534a94b",
		"0004": "3b21d7e7dfaa37e4998835016c19b663a1f0fae7d3c36a2f81d9bc59078c8c09",
		"0005": "4cc60fd5c3d653f5492c2aab03220f5a74f01a58ea3b8270b25b75dd03f930ed",
		"0006": "1c8e0e960f728662062bd73bf791bce7b6cee092cff099375fe4455660f0f7b9",
		"0007": "37c0378f6a4d18e98505e31baf50ebb8e36da071868beae05e4631a92fed351f",
	}
}

// GlobalCatalogHashBaseline binds descriptor paths and feature check hashes
// as well as the payload hashes. It must be updated only with an approved
// catalog change.
func GlobalCatalogHashBaseline() string {
	return "639a2e4ccc28afd91823fa8a0a98ac18b2b76816cab676610d66abc535be890a"
}
