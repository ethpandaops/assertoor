package checkconsensusblobcount

import "testing"

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr bool
	}{
		{
			name:   "defaults",
			mutate: func(*Config) {},
		},
		{
			name:   "max limit",
			mutate: func(c *Config) { c.BlobLimit = BlobLimitMax },
		},
		{
			name:    "unknown limit",
			mutate:  func(c *Config) { c.BlobLimit = "maximum" },
			wantErr: true,
		},
		{
			name:    "empty limit",
			mutate:  func(c *Config) { c.BlobLimit = "" },
			wantErr: true,
		},
		{
			name:    "zero block count would pass without checking a block",
			mutate:  func(c *Config) { c.BlockCount = 0 },
			wantErr: true,
		},
		{
			name:    "negative block count",
			mutate:  func(c *Config) { c.BlockCount = -1 },
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := DefaultConfig()
			test.mutate(&config)

			err := config.Validate()
			if test.wantErr && err == nil {
				t.Error("expected error, got none")
			}

			if !test.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
