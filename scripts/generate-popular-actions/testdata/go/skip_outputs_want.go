// Code generated by actionlint/scripts/generate-popular-actions. DO NOT EDIT.

package actionlint

// PopularActions is data set of known popular actions. Keys are specs (owner/repo@ref) of actions
// and values are their metadata.
var PopularActions = map[string]*ActionMetadata{
	"rhysd/action-setup-vim@v1": {
		Name: "Setup Vim",
		Inputs: ActionMetadataInputs{
			"neovim":  {"neovim", false},
			"token":   {"token", false},
			"version": {"version", false},
		},
		SkipOutputs: true,
	},
}

// OutdatedPopularActionSpecs is a spec set of known outdated popular actions. The word 'outdated'
// means that the runner used by the action is no longer available such as "node12".
var OutdatedPopularActionSpecs = map[string]struct{}{}
