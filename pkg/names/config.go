package names

type Config struct {
	InventoryYaml string            `yaml:"inventoryYaml" json:"inventoryYaml"`
	InventoryURL  string            `yaml:"inventoryUrl" json:"inventoryUrl"`
	Inventory     map[string]string `yaml:"inventory" json:"inventory"`
}
