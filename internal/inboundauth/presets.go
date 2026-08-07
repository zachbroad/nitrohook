package inboundauth

// PresetInfo describes a selectable provider preset for the UI dropdown.
type PresetInfo struct {
	Name           string `json:"name"`
	Label          string `json:"label"`
	NeedsSecret    bool   `json:"needs_secret"`
	NeedsPublicKey bool   `json:"needs_public_key"`
}

// presetConfigs holds the axis values for each preset (secrets filled in by the user).
var presetConfigs = map[string]SourceConfiguration{
	"github": {
		Scheme: SchemeHMAC, Preset: "github", Algo: "sha256", Encoding: "hex",
		SigHeader: "X-Hub-Signature-256", SigParser: "plain", SigPrefix: "sha256=", Template: "raw_body",
	},
	"forgejo": {
		Scheme: SchemeHMAC, Preset: "forgejo", Algo: "sha256", Encoding: "hex",
		SigHeader: "X-Forgejo-Signature", SigParser: "plain", Template: "raw_body",
	},
	"stripe": {
		Scheme: SchemeHMAC, Preset: "stripe", Algo: "sha256", Encoding: "hex",
		SigHeader: "Stripe-Signature", SigParser: "kv-comma", Template: "ts.body", TSToleranceS: 300,
	},
	"slack": {
		Scheme: SchemeHMAC, Preset: "slack", Algo: "sha256", Encoding: "hex",
		SigHeader: "X-Slack-Signature", SigParser: "slack", Template: "slack_v0",
		TSHeader: "X-Slack-Request-Timestamp", TSToleranceS: 300,
	},
	"shopify": {
		Scheme: SchemeHMAC, Preset: "shopify", Algo: "sha256", Encoding: "base64",
		SigHeader: "X-Shopify-Hmac-Sha256", SigParser: "plain", Template: "raw_body",
	},
	"standard-webhooks": {
		Scheme: SchemeHMAC, Preset: "standard-webhooks", Algo: "sha256", Encoding: "base64",
		SigHeader: "webhook-signature", SigParser: "space-list", Template: "id.ts.body",
		TSHeader: "webhook-timestamp", IDHeader: "webhook-id", TSToleranceS: 300,
	},
	"gitlab-token": {
		Scheme: SchemeToken, Preset: "gitlab-token", TokenHdr: "X-Gitlab-Token",
	},
	"discord-ed25519": {
		Scheme: SchemeEd25519, Preset: "discord-ed25519",
		SigHeader: "X-Signature-Ed25519", TSHeader: "X-Signature-Timestamp",
		TSToleranceS: 300,
	},
}

var presetMeta = []PresetInfo{
	{"github", "GitHub", true, false},
	{"forgejo", "Forgejo / Gitea", true, false},
	{"stripe", "Stripe", true, false},
	{"slack", "Slack", true, false},
	{"shopify", "Shopify", true, false},
	{"standard-webhooks", "Svix / Standard Webhooks", true, false},
	{"gitlab-token", "GitLab (token)", true, false},
	{"discord-ed25519", "Discord (Ed25519)", false, true},
}

// Preset returns a base Config for name (secrets left empty), or false if unknown.
func Preset(name string) (SourceConfiguration, bool) {
	cfg, ok := presetConfigs[name]
	return cfg, ok
}

// PresetNames returns UI metadata for all presets in display order.
func PresetNames() []PresetInfo {
	out := make([]PresetInfo, len(presetMeta))
	copy(out, presetMeta)
	return out
}
